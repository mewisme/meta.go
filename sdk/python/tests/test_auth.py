from __future__ import annotations

import pytest

from mewisme_meta import AppStateAuth, AppStateCookie, CookieAuth, Credentials


def test_credentials_require_exactly_one_second_factor() -> None:
    assert Credentials("user", "secret", otp="123456").otp == "123456"
    assert Credentials("user", "secret", totp="JBSWY3DPEHPK3PXP").totp == "JBSWY3DPEHPK3PXP"
    with pytest.raises(ValueError, match="exactly one of totp or otp"):
        Credentials("user", "secret")
    with pytest.raises(ValueError, match="exactly one of totp or otp"):
        Credentials("user", "secret", totp="totp-secret", otp="123456")


def test_auth_validation_rejects_malformed_sources_without_leaking_secrets() -> None:
    secret = "do-not-leak-this"
    with pytest.raises(ValueError) as caught:
        Credentials("", secret, otp="123456")
    assert secret not in str(caught.value)
    with pytest.raises(ValueError, match="cookies require c_user and xs"):
        CookieAuth({"c_user": "42"})
    with pytest.raises(ValueError, match="key and value"):
        AppStateCookie("", secret)
    with pytest.raises(ValueError, match="at least one cookie"):
        AppStateAuth(())
