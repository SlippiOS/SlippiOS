"""Slippi account sign-in using the same public web endpoints as slippi.gg."""

from __future__ import annotations

import json
import urllib.error
import urllib.request

# Public Firebase web configuration served to every browser by slippi.gg.
API_KEY = "AIzaSyAuQqc_wgqcUu3FqrICEPZ9Av_hPxMR_i4"
IDENTITY_URL = "https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword"
GRAPHQL_URL = "https://internal.slippi.gg/graphql"

USER_QUERY = """
query getUserKeyQuery($fbUid: String) {
  getUser(fbUid: $fbUid) {
    fbUid
    displayName
    connectCode { code }
    private { playKey }
  }
}
"""


class AuthError(Exception):
    pass


def _post(url: str, payload: dict, token: str | None = None) -> dict:
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    request = urllib.request.Request(
        url, data=json.dumps(payload).encode(), headers=headers
    )
    try:
        with urllib.request.urlopen(request, timeout=25) as response:
            return json.load(response)
    except urllib.error.HTTPError as exc:
        try:
            body = json.loads(exc.read())
            message = body.get("error", {}).get("message", "AUTH_FAILED")
        except (ValueError, AttributeError):
            message = "AUTH_FAILED"
        raise AuthError(_friendly(message)) from exc
    except (urllib.error.URLError, TimeoutError) as exc:
        raise AuthError("Cannot reach Slippi. Check the network connection.") from exc


def _friendly(code: str) -> str:
    return {
        "INVALID_LOGIN_CREDENTIALS": "Incorrect email or password.",
        "EMAIL_NOT_FOUND": "Incorrect email or password.",
        "INVALID_PASSWORD": "Incorrect email or password.",
        "USER_DISABLED": "This account has been disabled.",
        "TOO_MANY_ATTEMPTS_TRY_LATER": "Too many attempts. Try again later.",
    }.get(code, f"Slippi sign-in failed ({code}).")


def login(email: str, password: str) -> dict:
    """Return a user.json payload for the authenticated account."""
    session = _post(
        f"{IDENTITY_URL}?key={API_KEY}",
        {"email": email, "password": password, "returnSecureToken": True},
    )
    token = session.get("idToken")
    uid = session.get("localId")
    if not token or not uid:
        raise AuthError("Slippi sign-in failed.")

    result = _post(
        GRAPHQL_URL,
        {"query": USER_QUERY, "variables": {"fbUid": uid}},
        token=token,
    )
    user = (result.get("data") or {}).get("getUser")
    if not user:
        raise AuthError("No Slippi account data found for this login.")

    play_key = (user.get("private") or {}).get("playKey")
    connect_code = (user.get("connectCode") or {}).get("code")
    if not play_key:
        raise AuthError(
            "This account is not activated for online play. "
            "Activate it once at slippi.gg/online/enable."
        )

    return {
        "uid": uid,
        "displayName": user.get("displayName") or "",
        "connectCode": connect_code or "",
        "playKey": play_key,
    }
