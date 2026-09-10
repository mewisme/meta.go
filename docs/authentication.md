# Authentication and profiles

meta.go supports cookie-based sessions and credential login. Cookies are the simplest option when an authenticated browser session already exists.

## Cookies

```go
cookies, err := auth.ParseCookieString("c_user=...; xs=...")
if err != nil {
	return err
}
if err := cookies.ValidateRegular(); err != nil {
	return err
}
```

Browser-exported cookie JSON can be parsed with `auth.ParseBrowserCookieJSON`.

## Credential login

```go
login := auth.NewCredentialLogin()
cookies, err := login.Login(ctx, auth.Credentials{
	Identifier: "account@example.com",
	Password:   password,
	TOTP:       totpSecret,
})
```

`TOTP` is a TOTP seed and meta generates the current code when needed. Use `OTP` instead when a one-time numeric code has already been generated. `TOTP` and `OTP` are mutually exclusive.

Credential login uses the maintained login flow first and a compatibility flow only when the primary protocol changes. Authentication rejection, checkpoints, rate limits and protocol changes are returned as typed errors.

## Session validation

`auth.SessionValidator.Validate` checks an existing cookie session without requiring a password. `auth.SessionBootstrapper` exists for legacy token/bootstrap compatibility when those fields are explicitly needed.

## Profiles and secret storage

`auth.ProfileManager` separates profile metadata from secrets. Applications provide a `storage.ProfileStore` and `storage.SecretStore`.

Managed profile secrets include:

- cookies;
- session metadata;
- E2EE device state.

Renaming a profile is transactional across metadata and managed secrets. A partial copy/delete failure is rolled back instead of leaving two inconsistent profiles.

## Logout

Use `ProfileManager.Logout` with an explicit `LogoutPolicy`. meta.go does not guess which persisted state should be destroyed.
