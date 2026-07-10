package com.tradingplatform.api.security;

import io.jsonwebtoken.Claims;
import io.jsonwebtoken.JwtException;
import org.junit.jupiter.api.Test;

import java.time.Duration;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

// Pure unit test — no Spring context. Runs in milliseconds under surefire.
class JwtServiceTest {

    private static final String SECRET = "unit-test-secret-unit-test-secret-0123456789";

    private JwtService service(Duration accessTtl) {
        return new JwtService(SECRET, accessTtl, Duration.ofDays(7));
    }

    @Test
    void accessTokenRoundTripsClaims() {
        JwtService jwt = service(Duration.ofMinutes(15));
        String token = jwt.issueAccessToken(42L, "trader@example.com", "USER");

        Claims claims = jwt.parse(token);
        assertEquals("42", claims.getSubject());
        assertEquals("trader@example.com", claims.get("email", String.class));
        assertTrue(jwt.isAccessToken(claims));
        assertFalse(jwt.isRefreshToken(claims));
    }

    @Test
    void refreshTokenIsDistinguishableFromAccess() {
        JwtService jwt = service(Duration.ofMinutes(15));
        Claims claims = jwt.parse(jwt.issueRefreshToken(42L, "t@e.com", "USER"));
        assertTrue(jwt.isRefreshToken(claims));
        assertFalse(jwt.isAccessToken(claims));
    }

    @Test
    void weakSecretIsRejectedAtConstruction() {
        assertThrows(IllegalStateException.class,
                () -> new JwtService("too-short", Duration.ofMinutes(15), Duration.ofDays(7)));
    }

    @Test
    void expiredTokenFailsToParse() {
        JwtService jwt = service(Duration.ofSeconds(-1)); // already expired
        String token = jwt.issueAccessToken(1L, "t@e.com", "USER");
        assertThrows(JwtException.class, () -> jwt.parse(token));
    }

    @Test
    void tokenSignedWithAnotherSecretIsRejected() {
        JwtService issuer = service(Duration.ofMinutes(15));
        String token = issuer.issueAccessToken(1L, "t@e.com", "USER");

        JwtService other = new JwtService(
                "different-secret-different-secret-0123456789",
                Duration.ofMinutes(15), Duration.ofDays(7));
        assertThrows(JwtException.class, () -> other.parse(token));
    }
}
