package provider

// PasswordChangeContext contains all info about a password change event.
type PasswordChangeContext struct {
	Email    string
	Password string
	Role     string
}

// PasswordChangeHook is called after a successful local password change.
type PasswordChangeHook interface {
	OnPasswordChanged(ctx PasswordChangeContext) error
}
