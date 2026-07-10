package com.tradingplatform.api.auth;

import com.tradingplatform.api.auth.dto.LoginRequest;
import com.tradingplatform.api.auth.dto.RegisterRequest;
import com.tradingplatform.api.auth.dto.TokenResponse;
import com.tradingplatform.api.common.ApiException;
import com.tradingplatform.api.domain.Account;
import com.tradingplatform.api.domain.User;
import com.tradingplatform.api.repository.AccountRepository;
import com.tradingplatform.api.repository.UserRepository;
import com.tradingplatform.api.security.JwtService;
import io.jsonwebtoken.Claims;
import io.jsonwebtoken.JwtException;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

@Service
public class AuthService {

    private final UserRepository users;
    private final AccountRepository accounts;
    private final PasswordEncoder passwordEncoder;
    private final JwtService jwtService;

    public AuthService(UserRepository users, AccountRepository accounts,
                       PasswordEncoder passwordEncoder, JwtService jwtService) {
        this.users = users;
        this.accounts = accounts;
        this.passwordEncoder = passwordEncoder;
        this.jwtService = jwtService;
    }

    @Transactional
    public TokenResponse register(RegisterRequest request) {
        if (users.existsByEmail(request.email())) {
            throw ApiException.conflict("email already registered");
        }
        User user = users.save(
                new User(request.email(), passwordEncoder.encode(request.password())));
        // Every user gets an account; funding happens out of band for this scaffold.
        accounts.save(new Account(user.getId(), 0L));
        return tokensFor(user);
    }

    @Transactional(readOnly = true)
    public TokenResponse login(LoginRequest request) {
        User user = users.findByEmail(request.email())
                // Same error for unknown email and wrong password: don't reveal
                // which accounts exist.
                .orElseThrow(() -> ApiException.unauthorized("invalid credentials"));
        if (!passwordEncoder.matches(request.password(), user.getPasswordHash())) {
            throw ApiException.unauthorized("invalid credentials");
        }
        return tokensFor(user);
    }

    @Transactional(readOnly = true)
    public TokenResponse refresh(String refreshToken) {
        final Claims claims;
        try {
            claims = jwtService.parse(refreshToken);
        } catch (JwtException | IllegalArgumentException ex) {
            throw ApiException.unauthorized("invalid refresh token");
        }
        if (!jwtService.isRefreshToken(claims)) {
            throw ApiException.unauthorized("not a refresh token");
        }
        User user = users.findById(Long.valueOf(claims.getSubject()))
                .orElseThrow(() -> ApiException.unauthorized("invalid refresh token"));
        return tokensFor(user);
    }

    private TokenResponse tokensFor(User user) {
        String access = jwtService.issueAccessToken(user.getId(), user.getEmail(), user.getRole());
        String refresh = jwtService.issueRefreshToken(user.getId(), user.getEmail(), user.getRole());
        return TokenResponse.bearer(access, refresh);
    }
}
