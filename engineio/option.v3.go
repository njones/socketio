package engineio

import (
	"net/http"
	"time"
)

func WithPingInterval(d time.Duration) Option {
	return func(svr Server) {
		switch v := svr.(type) {
		case *serverV3:
			v.pingInterval = d
		}
	}
}

func WithUpgradeTimeout(d time.Duration) Option {
	return func(svr Server) {
		switch v := svr.(type) {
		case *serverV3:
			v.upgradeTimeout = d
		}
	}
}

func WithCookie(cookie http.Cookie) Option {
	return func(svr Server) {
		switch v := svr.(type) {
		case *serverV3:
			if cookie.Name != "" {
				v.cookie.name = cookie.Name
			}
			if cookie.Path != "" {
				v.cookie.path = cookie.Path
			}
			if cookie.HttpOnly {
				v.cookie.httpOnly = cookie.HttpOnly
			}

		case *serverV2:
			if cookie.Name != "" {
				v.cookie.name = cookie.Name
			}
			if cookie.Path != "" {
				v.cookie.path = cookie.Path
			}
			if cookie.HttpOnly {
				v.cookie.httpOnly = cookie.HttpOnly
			}
		}
	}
}
