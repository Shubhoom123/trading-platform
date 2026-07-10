package com.tradingplatform.api.security;

import io.jsonwebtoken.Claims;
import io.jsonwebtoken.Jwts;
import io.jsonwebtoken.security.Keys;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

import javax.crypto.SecretKey;
import java.nio.charset.StandardCharsets;
import java.time.Duration;
import java.time.Instant;
import java.util.Date;

// Issues and validates signed JWTs. Access tokens are short-lived; refresh
// tokens are long-lived and carry a "typ" claim so an access endpoint can never
// be satisfied by a refresh token (or vice versa).
@Service
public class JwtService {

    private static final String CLAIM_TYPE = "typ";
    private static final String TYPE_ACCESS = "access";
    private static final String TYPE_REFRESH = "refresh";

    private final SecretKey key;
    private final Duration accessTtl;
    private final Duration refreshTtl;

    public JwtService(
            @Value("${security.jwt.secret}") String secret,
            @Value("${security.jwt.access-token-ttl}") Duration accessTtl,
            @Value("${security.jwt.refresh-token-ttl}") Duration refreshTtl) {
        byte[] bytes = secret.getBytes(StandardCharsets.UTF_8);
        if (bytes.length < 32) {
            // HS256 needs >= 256 bits of key material. Fail fast rather than
            // silently signing with a weak key.
            throw new IllegalStateException(
                    "security.jwt.secret must be at least 32 bytes; set a strong JWT_SECRET");
        }
        this.key = Keys.hmacShaKeyFor(bytes);
        this.accessTtl = accessTtl;
        this.refreshTtl = refreshTtl;
    }

    public String issueAccessToken(Long userId, String email, String role) {
        return build(userId, email, role, TYPE_ACCESS, accessTtl);
    }

    public String issueRefreshToken(Long userId, String email, String role) {
        return build(userId, email, role, TYPE_REFRESH, refreshTtl);
    }

    private String build(Long userId, String email, String role, String type, Duration ttl) {
        Instant now = Instant.now();
        return Jwts.builder()
                .subject(String.valueOf(userId))
                .claim("email", email)
                .claim("role", role)
                .claim(CLAIM_TYPE, type)
                .issuedAt(Date.from(now))
                .expiration(Date.from(now.plus(ttl)))
                // Pin HS256 explicitly. jjwt otherwise picks the algorithm from
                // the key length (a >=64-byte secret would yield HS512), which
                // the Go gateway — a strict HS256 verifier — would reject.
                .signWith(key, Jwts.SIG.HS256)
                .compact();
    }

    public Claims parse(String token) {
        return Jwts.parser()
                .verifyWith(key)
                .build()
                .parseSignedClaims(token)
                .getPayload();
    }

    public boolean isAccessToken(Claims claims) {
        return TYPE_ACCESS.equals(claims.get(CLAIM_TYPE, String.class));
    }

    public boolean isRefreshToken(Claims claims) {
        return TYPE_REFRESH.equals(claims.get(CLAIM_TYPE, String.class));
    }
}
