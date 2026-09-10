# Authentication and profiles

meta.go supports cookies, AppState and credential login as explicit client authentication sources. Configure exactly one source per client.

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

## AppState

```go
state, err := auth.ParseAppStateJSON(appStateJSON)
if err != nil {
	return err
}
client, err := meta.NewClient(meta.WithAppState(state))
```

Canonical AppState items use `key` and `value`. The parser also accepts `name` as a compatibility alias for browser-exported cookie arrays. Duplicate keys use the last value.

## Credential login

```go
login := auth.NewCredentialLogin()
cookies, err := login.Login(ctx, auth.Credentials{
	Identifier: "account@example.com",
	Password:   password,
	TOTP:       totpSecret,
})
```

`TOTP` is a TOTP seed and meta generates the current code. Use `OTP` instead when a one-time numeric code has already been generated. Identifier, password and exactly one of `TOTP` or `OTP` are required.

Credential login may also be configured directly on the client:

```go
client, err := meta.NewClient(meta.WithCredentials(auth.Credentials{
	Identifier: "account@example.com",
	Password:   password,
	OTP:        otp,
}))
```

After successful credential resolution, the client retains only resulting cookies for reconnects and clears stored password/TOTP/OTP values.

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
