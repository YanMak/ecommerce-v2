package search_test

import (
	"bytes"
	"encoding/json"
	"io"
	"lesson2/adapters/inbound/httpapi"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"testing"
)

// Problem-style ошибка (как у respond.Problem), для парсинга негативов.
type problem struct {
	Code   string `json:"code"`
	Error  string `json:"error"`
	Errors []struct {
		Loc string `json:"loc"`
		Msg string `json:"msg"`
	} `json:"errors"`
}

// Мини-модель успеха: проверяем только ключевые поля, не всё эхо.
type echoResp struct {
	Path struct {
		TenantID string `json:"tenant_id"`
	} `json:"path"`
	Headers struct {
		AuthBearer string `json:"Authorization"`
		ClientVer  string `json:"X-Client-Version"`
		Locale     string `json:"X-Locale"` // en|ru
	} `json:"headers"`
	Cookie struct {
		SessionID string `json:"session_id"`
	} `json:"cookie"`
	Query struct {
		Limit int `json:"limit"`
	} `json:"query"`
}

func TestSearch_SmokeAndNegatives(t *testing.T) {
	//22 08 2025 Optional
	t.Parallel()

	// Один in-memory сервер на весь тест.
	ts := httptest.NewServer(httpapi.NewServer())
	defer ts.Close()

	validTenant := "550e8400-e29b-41d4-a716-446655440000"
	validSession := "8f14e45f-ea9b-4bf6-8a1d-0a9a9c2f3f71"
	bearer := "TOKEN"

	type req struct {
		tenant string
		q      url.Values
		h      map[string]string
		cookie string
		body   any
	}

	type exp struct {
		status  int
		wantLoc []string                       // ожидаемые loc'и в errors (для негативов)
		verify  func(t *testing.T, raw []byte) // доп. проверки тела (для успеха)
	}

	tests := map[string]struct {
		in  req
		out exp
	}{
		"ok": {
			in: req{
				tenant: validTenant,
				q: url.Values{
					"limit":        {"10"},
					"offset":       {"0"},
					"sort":         {"-created_at,name"},
					"created_from": {"2023-01-01T00:00:00Z"},
				},
				h: map[string]string{
					"Authorization":    "Bearer " + bearer,
					"X-Locale":         "en",
					"X-Client-Version": "1.2.3",
				},
				cookie: validSession,
				body: map[string]any{
					"q":            "nike",
					"category_ids": []string{"550e8400-e29b-41d4-a716-446655440000"},
					"price":        map[string]any{"min": 10.5, "max": 100},
					"tags":         []string{"men", "run"},
				},
			},
			out: exp{
				status: http.StatusOK,
				verify: func(t *testing.T, raw []byte) {
					var got echoResp
					if err := json.Unmarshal(raw, &got); err != nil {
						t.Fatalf("bad json: %v; raw=%s", err, raw)
					}
					if got.Path.TenantID != validTenant {
						t.Fatalf("path.tenant_id mismatch: %s", got.Path.TenantID)
					}
					if got.Cookie.SessionID != validSession {
						t.Fatalf("cookie.session_id mismatch: %s", got.Cookie.SessionID)
					}
					if got.Headers.AuthBearer != bearer {
						t.Fatalf("auth bearer mismatch: %s", got.Headers.AuthBearer)
					}
					if got.Query.Limit != 10 {
						t.Fatalf("query.limit mismatch: %d", got.Query.Limit)
					}
				},
			},
		},
		"bad path uuid": {
			in:  req{tenant: "not-a-uuid", h: map[string]string{"Authorization": "Bearer " + bearer}},
			out: exp{status: http.StatusBadRequest, wantLoc: []string{"path.tenant_id"}},
		},
		"missing bearer": {
			in:  req{tenant: validTenant, cookie: validSession, body: map[string]any{}},
			out: exp{status: http.StatusBadRequest, wantLoc: []string{"header.Authorization"}},
		},
		"missing cookie": {
			in:  req{tenant: validTenant, h: map[string]string{"Authorization": "Bearer " + bearer}},
			out: exp{status: http.StatusBadRequest, wantLoc: []string{"cookie.session_id"}},
		},
		"bad RFC3339": {
			in:  req{tenant: validTenant, cookie: validSession, q: url.Values{"created_from": {"yesterday"}}, h: map[string]string{"Authorization": "Bearer " + bearer}},
			out: exp{status: http.StatusBadRequest, wantLoc: []string{"query.created_from"}},
		},
		"limit out of range": {
			in:  req{tenant: validTenant, cookie: validSession, q: url.Values{"limit": {"0"}}, h: map[string]string{"Authorization": "Bearer " + bearer}},
			out: exp{status: http.StatusBadRequest, wantLoc: []string{"query.limit"}},
		},
		"unknown field in body": {
			in: req{
				tenant: validTenant,
				h:      map[string]string{"Authorization": "Bearer " + bearer},
				cookie: validSession,
				body:   map[string]any{"extra": 1},
			},
			out: exp{status: http.StatusBadRequest, wantLoc: []string{"body.extra"}},
		},
		"invalid type in body": {
			in: req{
				tenant: validTenant,
				h:      map[string]string{"Authorization": "Bearer " + bearer},
				cookie: validSession,
				body:   map[string]any{"price": map[string]any{"min": "oops", "max": 10}},
			},
			out: exp{
				status:  http.StatusBadRequest,
				wantLoc: []string{"body.price.min"}, // см. BindSearchBody: json.UnmarshalTypeError.Field → "min"
			},
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			// t.Parallel() — можно включить, если сервер stateless и нет shared-сайдэффектов.
			u := ts.URL + "/v1/tenants/" + tc.in.tenant + "/catalog:search"
			if len(tc.in.q) > 0 {
				u += "?" + tc.in.q.Encode()
			}

			var body io.Reader
			if tc.in.body != nil {
				var buf bytes.Buffer
				if err := json.NewEncoder(&buf).Encode(tc.in.body); err != nil {
					t.Fatalf("encode body: %v", err)
				}
				body = &buf
			}

			req, err := http.NewRequest(http.MethodPost, u, body)
			if err != nil {
				t.Fatalf("new request: %v", err)
			}
			// Заголовки
			for k, v := range tc.in.h {
				req.Header.Set(k, v)
			}
			if tc.in.body != nil && req.Header.Get("Content-Type") == "" {
				req.Header.Set("Content-Type", "application/json")
			}
			// Cookie
			if tc.in.cookie != "" {
				req.Header.Set("Cookie", "session_id="+tc.in.cookie)
			}

			resp, err := ts.Client().Do(req)
			if err != nil {
				t.Fatalf("do: %v", err)
			}
			defer resp.Body.Close()

			// Статус
			if resp.StatusCode != tc.out.status {
				raw, _ := io.ReadAll(resp.Body)
				t.Fatalf("status: want %d got %d; raw=%s", tc.out.status, resp.StatusCode, raw)
			}
			// Детерминизм транспорта
			assertHeaderPresent(t, resp, "X-Request-ID")
			if resp.StatusCode == http.StatusOK {
				assertHeaderContains(t, resp, "Content-Type", "application/json")
			}

			raw, _ := io.ReadAll(resp.Body)

			// Успех — выполняем verify()
			if resp.StatusCode == http.StatusOK {
				if tc.out.verify != nil {
					tc.out.verify(t, raw)
				}
				return
			}

			// Негатив — проверяем Problem и ожидаемые loc'и
			var pr problem
			if err := json.Unmarshal(raw, &pr); err != nil {
				t.Fatalf("bad problem json: %v; raw=%s", err, raw)
			}
			got := make([]string, 0, len(pr.Errors))
			for _, e := range pr.Errors {
				got = append(got, e.Loc)
			}
			sort.Strings(got)
			exp := append([]string(nil), tc.out.wantLoc...)
			sort.Strings(exp)
			if len(exp) > 0 && !sameStrings(got, exp) {
				t.Fatalf("error locs mismatch: want=%v got=%v; raw=%s", exp, got, raw)
			}
		})
	}

	t.Run("business invalid: limit too large", func(t *testing.T) {
		u := ts.URL + "/v1/tenants/" + validTenant + "/catalog:search?limit=99"

		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(map[string]any{}); err != nil {
			t.Fatalf("encode body: %v", err)
		}

		req, err := http.NewRequest(http.MethodPost, u, &buf)
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+bearer)
		req.Header.Set("Cookie", "session_id="+validSession)

		resp, err := ts.Client().Do(req)
		if err != nil {
			t.Fatalf("do: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			raw, _ := io.ReadAll(resp.Body)
			t.Fatalf("status: want 400 got %d; raw=%s", resp.StatusCode, raw)
		}
		// Транспортные заголовки на месте
		assertHeaderPresent(t, resp, "X-Request-ID")
		assertHeaderContains(t, resp, "Content-Type", "application/json")

		raw, _ := io.ReadAll(resp.Body)
		var pr problem
		if err := json.Unmarshal(raw, &pr); err != nil {
			t.Fatalf("bad problem json: %v; raw=%s", err, raw)
		}
		if pr.Code != "invalid" {
			t.Fatalf("code: want invalid, got %q; raw=%s", pr.Code, raw)
		}
		if pr.Error != "limit too large" {
			t.Fatalf("error: want %q, got %q; raw=%s", "limit too large", pr.Error, raw)
		}
		if len(pr.Errors) != 0 {
			t.Fatalf("unexpected field errors: %+v; raw=%s", pr.Errors, raw)
		}
	})
}

// --- helpers ---

func assertHeaderPresent(t *testing.T, resp *http.Response, key string) {
	t.Helper()
	if resp.Header.Get(key) == "" {
		t.Fatalf("missing header %q", key)
	}
}
func assertHeaderContains(t *testing.T, resp *http.Response, key, substr string) {
	t.Helper()
	if v := resp.Header.Get(key); v == "" || !contains(v, substr) {
		t.Fatalf("header %q = %q, want contains %q", key, v, substr)
	}
}
func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || (len(s) > len(sub) && (indexOf(s, sub) >= 0)))
}
func indexOf(s, sub string) int {
	// без strings.Index, чтобы держать минимум импортов — можно заменить на strings.Index
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
