package gengo

import (
	"strings"
	"testing"
)

func TestNormalizeGoAcronyms(t *testing.T) {
	src := []byte(`package contract
type Message struct {
	UserId string
	UserIds []string
	IconUrl string
	Ok bool
	IdempotencyKey string
}
func (m *Message) GetUserId() string { return m.UserId }
`)

	got, err := normalizeGoAcronyms(src)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	text := string(got)
	for _, want := range []string{"\tUserID ", "\tUserIDs ", "\tIconURL ", "\tOK ", "GetUserID() string", "m.UserID"} {
		if !strings.Contains(text, want) {
			t.Errorf("normalized source does not contain %q:\n%s", want, text)
		}
	}
	if !strings.Contains(text, "IdempotencyKey string") {
		t.Errorf("non-suffix Id was changed:\n%s", text)
	}
}
