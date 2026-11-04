package functions

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestHandleHTTPNames(t *testing.T) {
	originalPasswords := usersPasswords
	usersPasswords = map[string]string{"test": "test"}
	t.Cleanup(func() { usersPasswords = originalPasswords })

	tests := []struct {
		name  string
		valid bool
	}{
		{"eve.games", true},
		{"eve.games-helixcore-useast1", true},
		{"Customer_123-prod", true},
		{"", false},
		{".", false},
		{"..", false},
		{"../outside", false},
		{"a/b", false},
		{`a\b`, false},
		{"a b", false},
		{"a\nb", false},
	}
	for _, test := range tests {
		for _, parameter := range []string{"customer", "instance"} {
			t.Run(parameter+"/"+test.name, func(t *testing.T) {
				query := url.Values{"customer": {"eve.games"}, "instance": {"eve.games-helixcore-useast1"}}
				query.Set(parameter, test.name)
				request := httptest.NewRequest(http.MethodPost, "/json/?"+query.Encode(), nil)
				request.SetBasicAuth("test", "test")
				response := httptest.NewRecorder()
				customer, instance, err := HandleHTTP(response, request, logrus.New(), t.TempDir())
				if test.valid {
					if err != nil || customer != query.Get("customer") || instance != query.Get("instance") {
						t.Fatalf("HandleHTTP() = (%q, %q, %v)", customer, instance, err)
					}
				} else if err == nil || response.Code != http.StatusBadRequest {
					t.Fatalf("HandleHTTP() error = %v, status = %d; want error and 400", err, response.Code)
				}
			})
		}
	}
}
