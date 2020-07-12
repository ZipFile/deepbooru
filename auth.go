package deepbooru

type AccessLevel int8

const (
	LevelAnonymous AccessLevel = iota
	LevelUser
	LevelPowerUser
	LevelMod
	LevelAdmin
)

type Auth struct {
	ID    string
	Name  string
	Level AccessLevel
}

var Anonymous = Auth{"", "anonymous", LevelAnonymous}

type Authorizer interface {
	Authorize(credentials string) (Auth, error)
}

type AuthorizerFunc func(credentials string) (Auth, error)

func (f AuthorizerFunc) Close() {}

func (f AuthorizerFunc) Authorize(credentials string) (Auth, error) {
	return f(credentials)
}

func NoAuth(credentials string) (Auth, error) {
	return Anonymous, nil
}

func NoopAuthorizer() Authorizer {
	return AuthorizerFunc(NoAuth)
}
