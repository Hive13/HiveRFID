package intweb

// This package is a partial implementation of
// https://wiki.hive13.org/view/Access_Protocol. As this protocol is
// considered sort of legacy at this point, it does not put
// extraordinary effort into correctness - so some things are ignored,
// such as verifying the checksums on communications with the server.

import (
	"bytes"
	"fmt"
	"log"
	"crypto/sha512"
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"net/http"
)

// Session contains parameters for intweb communications.
//
// "Session" is something of a misnomer as this struct contains no
// state, though conceivably it could present a nicer interface by
// doing so (e.g. for things like the nonce the server uses).
type Session struct {
	// The name of the device
	Device string
	// The device key
	DeviceKey []byte
	// The URL of the intweb server, including /api/access.
	URL string
	// Set true for more verbose logging
	Verbose bool
	// The HTTP client (if a custom one is needed)
	Client *http.Client
}

// PostIntweb POSTs a message to the intweb server, returning the reply.
//
// Having a proper message format, including nonces and checksums, is
// up to the caller. This call does not do it.
func (s *Session) PostIntweb(data interface {}) ([]byte, error) {
	msg_json, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	if s.Verbose {
		log.Printf("Request: POST to %s: %s", s.URL, msg_json)
	}

	resp, err := s.Client.Post(s.URL, "application/json", bytes.NewBuffer(msg_json))
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if s.Verbose {
		log.Printf("Response: HTTP %d, %s", resp.StatusCode, body)
	}
	if resp.StatusCode != 200 {
		return nil, &Error{
			Msg: fmt.Sprintf("HTTP code %d", resp.StatusCode),
			Resp: nil,
		}
	}
	if err != nil {
		return nil, err
	}
	
	return body, nil
}

// GetNonce requests a new nonce from the server.
//
// This is a necessary first step for many other requests.
func (s *Session) GetNonce() (string, error) {
	d := MessageData{
		Operation: "get_nonce",
		Version: 2,
		RandomResponse: randomResponse(),
	}
	cs, err := checksum(s.DeviceKey, d)
	if err != nil {
		return "", err
	}
	
	msg := map[string](interface {}){
		"data": d,
		"device": s.Device,
		"checksum": fmt.Sprintf("%X", cs),
	}

	resp_bytes, err := s.PostIntweb(msg)
	if err != nil {
		return "", err
	}

	var resp Response
	err = json.Unmarshal(resp_bytes, &resp)
	if err != nil {
		return "", err
	}
	data, err := DecodeAndCheck(&resp)
	if err != nil {
		return "", err
	}

	return data.NewNonce, nil
}

// Access requests access to some item for some badge number.
//
// Item and badge number must be in exactly the same format as in the
// intweb database.  Nonce must also be supplied, e.g. from a previous
// GetNonce() call.
func (s *Session) Access(nonce string, item string, badge uint64) (bool, string, error) {

	d := AccessReqData{
		Operation: "access",
		Version: 2,
		RandomResponse: randomResponse(),
		Nonce: nonce,
		Item: item,
		Badge: badge,
	}
	cs, err := checksum(s.DeviceKey, d)
	if err != nil {
		return false, "", err
	}
	
	msg := map[string](interface {}){
		"data": d,
		"device": s.Device,
		"checksum": fmt.Sprintf("%X", cs),
	}

	resp_bytes, err := s.PostIntweb(msg)
	if err != nil {
		return false, "", err
	}

	var resp Response
	err = json.Unmarshal(resp_bytes, &resp)
	if err != nil {
		return false, "", err
	}
	data, err := DecodeAndCheck(&resp)
	if err != nil {
		return false, "", err
	}

	return data.Access, data.Error, nil
}

// MessageData contains the data for a generic message that is sent to
// intweb, e.g. to request a new nonce.
type MessageData struct {
	Operation      string `json:"operation"`
	RandomResponse []int  `json:"random_response"`
	Version        int    `json:"version"`
	// These fields must remain in sorted order for the checksum.
}

// AccessReqData contains the data for an access request message that
// is sent to intweb.
type AccessReqData struct {
	Badge          uint64 `json:"badge"` 
	Item           string `json:"item"`
	Nonce          string `json:"nonce"`
	Operation      string `json:"operation"`
	RandomResponse []int  `json:"random_response"`
	Version        int    `json:"version"`
	// These fields must remain in sorted order for the checksum.
	// Ordinarily I would have just embedded MessageData.
}

// Response is a catch-all structure for a response from intweb.
//
// In theory, we could parse this in different ways depending on which
// message type it is.  In practice, the documentation is hairy and
// the server-side implementation is even hairier, so I really don't
// care.
type Response struct {
	Data     json.RawMessage `json:"data"`
	Response *bool           `json:"response"`
	Version  string          `json:"version"`
}

// RespData contains the data from a few intweb response types.
//
// This includes a generic error reply, a reply to requesting a new
// nonce, and a reply to an access request.  Not all fields will
// always be used.
//
// See the DecodeAndCheck function for both producing an instance of
// this, and checking some of the common errors
type RespData struct {
	NewNonce       string `json:"new_nonce"`
	NonceValid     bool   `json:"nonce_valid"`
	Random         []int  `json:"random"`
	RandomResponse []int  `json:"random_response"`
	Response       bool   `json:"response"`
	// Disabled to work around a server work-around:
	//Version        string `json:"version"`
	Data           string `json:"data"`
	Access         bool   `json:"access"`
	Error          string `json:"error"`
}

// An error reported by the intweb server.
type Error struct {
	// The message text (which may be directly from the server, or may
	// be a summary from some flag that is set):
	Msg string
	// The actual response that produced this error:
	Resp *RespData
}

func (err *Error) Error() string {
	return fmt.Sprintf("intweb reported error: %s", err.Msg)
}

// DecodeAndCheck attempts to parse a Response into RespData.
//
// Returns an error if this fails. Errors may be from JSON
// unmarshaling, or may be an intweb.Error.
func DecodeAndCheck(r *Response) (*RespData, error) {

	if r.Response != nil && !(*r.Response) {
		var s string
		if err := json.Unmarshal(r.Data, &s); err != nil {
			return nil, err
		}
		return nil, &Error{ Msg: s, Resp: nil }
	}
	
	var data RespData
	err := json.Unmarshal(r.Data, &data)
	if err != nil {
		return nil, err
	}

	if !data.Response {
		return nil, &Error{ Msg: data.Data, Resp: &data }
	}
	
	if !data.NonceValid {
		return nil, &Error{ Msg: "Nonce invalid", Resp: &data }
	}

	// TODO: Check checksums and random response?

	return &data, nil
}

// randomResponse returns an array with 16 random values (0-255).
func randomResponse() []int {
	resp := make([]int, 16)
	for i, _ := range resp {
		resp[i] = rand.Intn(256)
	}
	return resp
}

// checksumRaw returns the SHA-512 checksum for a given device key and data.
func checksumRaw(key []byte, data []byte) []byte  {
	h := sha512.New()
	h.Write(key)
	h.Write(data)
	return h.Sum(nil)
}

// checksum returns the SHA-512 checksum for some device key, and JSON data.
//
// This attempts to turn 'data' to JSON, and then computes the
// checksum over the device key and this data. 
func checksum(key []byte, data interface{}) ([]byte, error) {
	data_json, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return checksumRaw(key, data_json), nil
}

