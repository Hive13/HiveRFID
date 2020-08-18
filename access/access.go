package access

// access is the high-level service that connects the RFID reader,
// intweb, and HTTP server.

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/stianeikeland/go-rpio/v4"
	
	"hive13/rfid/intweb"
	"hive13/rfid/wiegand"
)

const (
	// URL to use for door opening:
	open_door_url = "/open_door"
	// Form key for badge number:
	open_door_key_badge = "badge"
)

type Config struct {
	// Pin number for Wiegand D0 of the badge reader (as GPIO/BCM pin):
	PinD0 int
	// Pin number for Wiegand D1 of the badge reader (as GPIO/BCM pin):
	PinD1 int
	// Pin number for the badge reader's beeper pin (as GPIO/BCM pin):
	PinBeeper int
	// Pin number for the badge reader's LED pin (as GPIO/BCM pin):
	PinLED int
	// Pin number to control door lock/latch relay (as GPIO/BCM pin):
	PinLock int
	// Time to hold PinLock high before bringing it back low:
	LockHoldTime time.Duration
	// URL for intweb, including /api/access
	IntwebURL string
	// Device name for intweb
	IntwebDevice string
	// Device key for intweb
	IntwebDeviceKey []byte
	// Item to try to access
	IntwebItem string
	// Length of time to keep a badge in cache for (starting from its
	// last use):
	BadgeCacheTime time.Duration
	// Address for HTTP server to listen on
	ListenAddr string
	// True to log more verbosely (e.g. all HTTP POSTs & replies)
	Verbose bool
}

// Some state/context for HTTP server:
type ServerCtx struct {
	Cfg *Config

	// A door-open request will be sent on this channel:
	OpenRqs chan<- HTTPOpenRequest

	// Initialized pin to control door latch:
	Lock rpio.Pin

	// Timer which, upon expiration, will trigger the door latch being
	// locked again.  Upon every lock, this should have .Stop() and
	// .Reset() called.
	ReLockTimer *time.Timer

	// Cached badges. Key = badge number, value = time at which to
	// expire this badge.
	Cache map[uint64]time.Time
}

// HTTPOpenRequest is a request to open the door (received via HTTP).
//
// Whoever receives this request must send something back over 'Reply'
// - either a nil if it processed the request successfully, or else an
// error for why it would not be.  Once this is done, the entire
// channel should be closed and the member set to nil.
type HTTPOpenRequest struct {
	// The badge number 
	Badge uint64
	Reply chan<- error
}

// Generic error class for door access being denied:
type AccessDeniedError struct {
	Msg string
}
func (a AccessDeniedError) Error() string {
	return a.Msg
}

func Run(cfg *Config) {
	beep_pin := rpio.Pin(cfg.PinBeeper)
	led_pin := rpio.Pin(cfg.PinLED)
	lock_pin := rpio.Pin(cfg.PinLock)
	if err := rpio.Open(); err != nil {
		log.Fatal(err)
	}
	defer rpio.Close()
	beep_pin.Output()
	led_pin.Output()
	for x := 0; x < 5; x++ {
		beep_pin.Toggle()
		led_pin.Toggle()
		time.Sleep(time.Millisecond * 20)
	}
	beep_pin.High()
	led_pin.High()

	// Make sure that both are off (they're active-low) when we exit:
	defer beep_pin.High()
	defer led_pin.High()
	// And likewise for lock, which is active-high:
	defer lock_pin.Low()
	
	log.Printf("Listening for badges...")
	badges, err := wiegand.ListenBadges(cfg.PinD0, cfg.PinD1)
	if err != nil {
		log.Fatal(err)
	}

	s := intweb.Session{
		Device: cfg.IntwebDevice,
		DeviceKey: cfg.IntwebDeviceKey,
		URL: cfg.IntwebURL,
		Verbose: cfg.Verbose,
		Client: &http.Client{
			// Avoid transient network issues blocking forever:
			Timeout: 15 * time.Second,
		},
	}
	log.Printf("Using intweb device: %s", s.Device)
	log.Printf("Using URL: %s", s.URL)

	http_rqs := make(chan HTTPOpenRequest)

	// Set up re-lock timer:
	relock := time.AfterFunc(cfg.LockHoldTime, func() {
		if cfg.Verbose {
			log.Printf("Closing lock.")
		}
		lock_pin.Low()
	})
	// We don't want it to trigger yet:
	relock.Stop()
	// We'll call .Stop() & .Reset() every time we unlock.  This way,
	// it's always the *last* unlock that sets the delay, and repeated
	// unlocks inside that delay don't trigger repeated re-locks.
	
	// Start HTTP server and supply some state:
	ctx := ServerCtx{
		Cfg: cfg,
		OpenRqs: http_rqs,
		Lock: lock_pin,
		ReLockTimer: relock,
		Cache: make(map[uint64]time.Time),
	}
	http.HandleFunc(open_door_url, ctx.open_door_handler)
	go func() {
		srv := &http.Server{
			Addr: cfg.ListenAddr,
			ReadTimeout: 20 * time.Second,
			WriteTimeout: 20 * time.Second,
		}
		log.Printf("Starting HTTP server on %s...", cfg.ListenAddr)
		log.Fatal(srv.ListenAndServe())
	}()

	cache_expire := make(chan uint64)
	
	// We now have two channels that receive request to open the door:
	// 'badges' for badge scans, 'http_rqs' for HTTP requests.
	// Monitor both. They intentionally block each other.
	log.Printf("Starting main loop...")
	for {
		select {
		// Badge scan:
		case v := <-badges:
			if cfg.Verbose {
				log.Printf("Main loop: Received badge: %+v", v)
			}

			if !v.LengthOK {
				if cfg.Verbose {
					log.Printf("Main loop: Badge has wrong number of bits, ignoring")
				}
				break
			}
			if !v.ParityOK {
				if cfg.Verbose {
					log.Printf("Main loop: Badge checksum mismatch, ignoring")
				}
				break
			}

			badge := v.Value
			log.Printf("Main loop: Received badge %d", badge)

			_, err := ctx.handle_badge(&s, badge, cache_expire)
			if err != nil {
				log.Printf("%+v", err)
			}

		// Incoming HTTP request:
		case rq := <-http_rqs:
			badge := rq.Badge

			log.Printf("Main loop: Received HTTP request for badge %+v", badge)

			_, err := ctx.handle_badge(&s, badge, cache_expire)
			rq.Reply <- err
			close(rq.Reply)

		// While idle, blink LED and scrub cache if needed:
		case <-time.After(1000 * time.Millisecond):
			ctx.scrub_cache()
			go func() {
				led_pin.Low()
				<-time.After(50 * time.Millisecond)
				led_pin.High()
			}()
			
		// Expire cache entries from background requests as-needed:
		case badge := <-cache_expire:
			log.Printf("Main loop: Removed badge %+v from cache (denied access in background)", badge)
			delete(ctx.Cache, badge)
		}
	}
}

// Handle door-open request (whether from badge reader or from HTTP).
//
// This returns: (access allowed, error).
//
// If error is non-nil, something prevented access from even being
// checked.  If error is nil, but access is false, then an intweb call
// denied access to this badge.  If error is nil and access is true,
// then access was allowed either by an intweb call or by the badge
// already being cached.
//
// If access is true, but the badge was cached, then a goroutine is
// started which checks the badge with intweb in the background.  If
// access for this badge is denied, the badge is sent over
// 'cache_expire'. (The point of this is so the main loop can safely
// clear a badge entry out of the cache if it was denied access.)
//
// Cache is always updated if there is no error. A badge that is
// granted access always has its cache expiration updated. A badge
// that is denied access always has its cache entry removed.
func (ctx *ServerCtx) handle_badge(s *intweb.Session, badge uint64,
	cache_expire chan<- uint64) (bool, error) {

	access := false
	var why string
	var err error

	check_intweb := func() (bool, string, error) {
		nonce, err := s.GetNonce()
		if err != nil {
			log.Printf("handle_badge: Failed to get nonce, %s", err)
			return false, "", err
		}

		access, why, err = s.Access(nonce, ctx.Cfg.IntwebItem, badge)
		if err != nil {
			log.Printf("handle_badge: Access request failed, %s", err)
			return false, "", err
		}

		return access, why, nil
	}

	// Check cache first:
	_, access_cache := ctx.Cache[badge]
	if access_cache {
		// If it was in the cache, then call check_intweb - but in the
		// background:
		log.Printf("handle_badge: Badge %+v is in cache", badge)
		access = true
		go func() {
			acc2, _, err2 := check_intweb()
			if err2 == nil && !acc2 {
				cache_expire <- badge
			}
		}()
	} else {
		// If it wasn't in the cache, then check intweb now:
		if access, why, err = check_intweb(); err != nil {
			return false, err
		}
	}

	ctx.Cache[badge] = time.Now().Add(ctx.Cfg.BadgeCacheTime)

	if !access {
		log.Printf("handle_badge: Removed badge %+v from cache (denied access)",
			badge)
		delete(ctx.Cache, badge)
		err = AccessDeniedError{ why }
	}
	
	ctx.handle_access(access, badge, why)
	
	return access, err
}


// HTTP handler for a request to /open_door:
func (ctx *ServerCtx) open_door_handler(w http.ResponseWriter, r *http.Request) {
	// Various sanity checks:
	if r.Method != "POST" {
		log.Printf("%s: Unsupported HTTP %s", open_door_url, r.Method)
		http.Error(w, "Method is not supported.", http.StatusNotFound)
		return
	}
	
	if err := r.ParseForm(); err != nil {
		errstr := fmt.Sprintf("Error parsing form: %s", err)
		log.Printf("%s: %s", open_door_url, errstr)
		http.Error(w, errstr, http.StatusBadRequest)
		return
	}

	badges, ok := r.Form[open_door_key_badge]
	if !ok {
		errstr := fmt.Sprintf("Form key '%s' is missing", open_door_key_badge)
		log.Printf("%s: %s", open_door_url, errstr)
		http.Error(w, errstr, http.StatusBadRequest)
		return
	}
	
	badge, err := strconv.ParseUint(badges[0], 10, 0)
	if err != nil {
		errstr := fmt.Sprintf("Error parsing badge: %s", err)
		log.Printf("%s: %s", open_door_url, errstr)
		http.Error(w, errstr, http.StatusBadRequest)
		return
	}

	// Finally, turn this to a request for the main loop:
	err_ch := make(chan error)
	rq := HTTPOpenRequest{
		Badge: badge,
		Reply: err_ch,
	}

	// Attempt to send the request to the main loop (which might be
	// busy handling something else):
	log.Printf("%s: Got badge %d, sending request to main loop...",
		open_door_url, badge)
	select {
	case ctx.OpenRqs <- rq:
		// Do nothing else - main loop read our request.
	case <-time.After(15 * time.Second):
		errstr := fmt.Sprintf("Timed out waiting on main loop")
		log.Printf("%s: %s", open_door_url, errstr)
		http.Error(w, errstr, http.StatusServiceUnavailable)
		return
	}

	// Wait around for the main loop's reply:
	select {
	case err := <-err_ch:
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
	case <-time.After(30 * time.Second):
		// This shouldn't ever happen. HTTP requests should always
		// time out and throw an error.
		errstr := fmt.Sprintf("Main loop received request, but didn't reply?")
		log.Printf("%s: %s", open_door_url, errstr)
		http.Error(w, errstr, http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "OK")
}

// handle_access handles a allowed/denied request for access.
//
// The parameter 'access' is true if access was allowed, and false if
// denied. That is: All necessary authentication/authorization has
// already been done, but it is up to this function to execute
// something on this decision - like flipping the door lock, or
// printing some kind of error.
//
// 'why' is set only if 'access' is false, and supplies a reason why
// access was denied.
func (ctx *ServerCtx) handle_access(access bool, badge uint64,
	why string) error {

	if access {
		log.Printf("Access allowed for %d!", badge)
		if ctx.Cfg.Verbose {
			log.Printf("Opening lock for %s...", ctx.Cfg.LockHoldTime)
		}

		ctx.Lock.High()
		
		ctx.ReLockTimer.Stop()
		ctx.ReLockTimer.Reset(ctx.Cfg.LockHoldTime)
	} else {
		log.Printf("Access denied for %d (why: %s)", badge, why)
	}

	return nil
}

// scrub_cache removes expired entries in the badge cache.  It returns
// the number of entries removed.
func (ctx *ServerCtx) scrub_cache() int {

	now := time.Now()
	to_del := make(map[uint64]bool)
	
	for badge, expiration := range ctx.Cache {
		if now.After(expiration) {
			log.Printf("scrub_cache: Expiring badge %+v", badge)
			to_del[badge] = true
		}
	}

	for badge, _ := range to_del {
		delete(ctx.Cache, badge)
	}

	return len(to_del)
}
