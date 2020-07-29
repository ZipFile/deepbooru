package deepbooru

import (
	"errors"
	"reflect"
	"testing"
)

func TestAuthorizerFuncAuthorize(t *testing.T) {
	expectedAuth := Auth{"id", "test", LevelPowerUser}
	testErr := errors.New("test")

	var authorizer AuthorizerFunc = func(credentials string) (Auth, error) {
		return Auth{"id", credentials, LevelPowerUser}, testErr
	}

	auth, err := authorizer.Authorize("test")

	if !reflect.DeepEqual(auth, expectedAuth) {
		t.Errorf("auth: %#v; expected: %#v", auth, expectedAuth)
	}

	if err != testErr {
		t.Errorf("err: %s; expected: %s", err, testErr)
	}
}

func TestNoopAuthorizer(t *testing.T) {
	expectedAuth := Auth{"", "anonymous", LevelAnonymous}
	authorizer := NoopAuthorizer()
	auth, err := authorizer.Authorize("test")

	if !reflect.DeepEqual(auth, expectedAuth) {
		t.Errorf("auth: %#v; expected: %#v", auth, expectedAuth)
	}

	if err != nil {
		t.Errorf("err: %s; expected: nil", err)
	}
}
