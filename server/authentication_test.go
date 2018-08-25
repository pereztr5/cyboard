package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/pereztr5/cyboard/server/apptest"
	"github.com/stretchr/testify/assert"
)

func loginReq() (http.ResponseWriter, *http.Request) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/login", nil)
	r.Form = make(url.Values)
	r.Form.Add(formCredsTeam, "team1")
	r.Form.Add(formCredsPass, "ramjam")
	return w, r
}

func TestCheckCreds(t *testing.T) {
	// Setup Database
	apptest.PrepDatabase(t)

	tests := map[string]struct {
		formPrep func(f *url.Values)
		expect   bool
	}{
		"valid": {
			formPrep: func(f *url.Values) {},
			expect:   true,
		},
		"missing password": {
			formPrep: func(f *url.Values) {
				f.Del(formCredsPass)
			},
			expect: false,
		},
		"missing teamname": {
			formPrep: func(f *url.Values) {
				f.Del(formCredsTeam)
			},
			expect: false,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			w, r := loginReq()
			tt.formPrep(&r.Form)
			loginState := CheckCreds(w, r)
			assert.Equal(t, tt.expect, loginState)
		})
	}
}
