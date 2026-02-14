package provider

// PasswordChangeHook is called after a successful local password change.
type PasswordChangeHook interface {
	OnPasswordChanged(email, newPassword string) error
}
