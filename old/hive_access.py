#!/usr/bin/env python3
#########################################################################
# Scratch code to authenticate with intweb device access
# Authors: Chris Hodapp
# 2020-01-19, Hive13 Cincinnati
#########################################################################

# See:
# https://wiki.hive13.org/view/Access_Protocol
# https://github.com/Hive13/hive-rfid-door-controller

import json
import random
import hashlib
import subprocess

import requests

# DEVICE is string, DEVICE_KEY is bytestring
from creds import DEVICE, DEVICE_KEY

URL = "https://intweb.at.hive13.org/api/access"
RFID_LISTENER_BIN = "../../rpi-wiegand-reader/reader.o"

def get_random_response(size=16):
    return [random.randint(0, 255) for _ in range(size)]

def get_checksum(key, data):
    s = json.dumps(data, sort_keys=True, separators=(",", ":")).encode()
    print(s)
    m = hashlib.sha512()
    m.update(key)
    m.update(s)
    return m.hexdigest().upper()

def get_nonce(device, device_key):
    msg = {
        "data": {
            "operation": "get_nonce",
            "version": 2,
            "random_response": get_random_response(),
        },
        "device": device,
    }
    cs = get_checksum(device_key, msg["data"])
    msg["checksum"] = cs

    print("Posting: {}".format(msg))
    res = requests.post(URL, json = msg)
    print(res)
    if res.status_code != requests.codes.ok:
        raise Exception("Received HTTP error")
    print("Status code OK")
    print(res.json())
    d = res.json()
    err = d["data"].get("error", None)
    if err is not None:
        raise Exception("Server reported error: {}".format(err))
    return d["data"]["new_nonce"]

def get_access(device, device_key, nonce, item, badge):
    msg = {
        "data": {
            "operation": "access",
            "version": 2,
            "random_response": get_random_response(),
            "nonce": nonce,
            "item": item,
            "badge": badge,
        },
        "device": device,
    }
    cs = get_checksum(device_key, msg["data"])
    msg["checksum"] = cs
    print("Posting: {}".format(msg))
    res = requests.post(URL, json = msg)
    print(res)
    if res.status_code != requests.codes.ok:
        print(res.text)
        raise Exception("Received HTTP error")
    d = res.json()
    print(d)
    if not d["data"]["nonce_valid"]:
        raise Exception("Server reported invalid nonce")
    err = d["data"].get("error", None)
    if err is not None:
        raise Exception("Server reported error: {}".format(err))
    return d

def main():
    with subprocess.Popen([RFID_LISTENER_BIN], stdout=subprocess.PIPE) as proc:
        for line in proc.stdout:
            line = line.strip()
            ts,count,_,ok, *rest = line.split(b",")
            if ok != b"OK":
                print("Ignored line: {}".format(line))
                continue
            badge = int(rest[0])
            print("Badge number: {}".format(badge))
            nonce = get_nonce(DEVICE, DEVICE_KEY)
            print("New nonce: {}".format(nonce))
            resp = get_access(DEVICE, DEVICE_KEY, nonce, "main_door", 1052895) # 7515388)
            print(resp)
            access = resp.get("data", {}).get("access", False)
            if access:
                print("-"*60)
                print("badge {} authenticaorizated".format(badge))
                print("dO mAgIc HeRe tO oPeN dOoR!!!1111111111111111111111111111")
                print("-"*60)
            else:
                print("Access denied :(")

if __name__ == "__main__":
    main()
