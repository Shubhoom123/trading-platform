package com.tradingplatform.api.security;

import io.jsonwebtoken.Claims;
import io.jsonwebtoken.JwtException;
import jakarta.servlet.FilterChain;
import jakarta.servlet.ServletException;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import org.springframework.lang.NonNull;
import org.springframework.security.authentication.UsernamePasswordAuthenticationToken;
import org.springframework.security.core.authority.SimpleGrantedAuthority;
import org.springframework.security.core.context.SecurityContextHolder;
import org.springframework.security.web.authentication.WebAuthenticationDetailsSource;
import org.springframework.stereotype.Component;
import org.springframework.web.filter.OncePerRequestFilter;

import java.io.IOException;
import java.util.List;

// Validates the Bearer access token on each request and populates the security
// context. Only access tokens are accepted here — a refresh token cannot be
// used to call a protected endpoint.
@Component
public class JwtAuthenticationFilter extends OncePerRequestFilter {

    private static final String HEADER = "Authorization";
    private static final String PREFIX = "Bearer ";

    private final JwtService jwtService;

    public JwtAuthenticationFilter(JwtService jwtService) {
        this.jwtService = jwtService;
    }

    @Override
    protected void doFilterInternal(@NonNull HttpServletRequest request,
                                    @NonNull HttpServletResponse response,
                                    @NonNull FilterChain chain)
            throws ServletException, IOException {

        String header = request.getHeader(HEADER);
        if (header == null || !header.startsWith(PREFIX)) {
            chain.doFilter(request, response);
            return;
        }

        String token = header.substring(PREFIX.length());
        try {
            Claims claims = jwtService.parse(token);
            if (jwtService.isAccessToken(claims)
                    && SecurityContextHolder.getContext().getAuthentication() == null) {

                Long userId = Long.valueOf(claims.getSubject());
                String email = claims.get("email", String.class);
                String role = claims.get("role", String.class);
                var principal = new AuthenticatedUser(userId, email, role);

                var authorities = List.of(new SimpleGrantedAuthority("ROLE_" + role));
                var auth = new UsernamePasswordAuthenticationToken(principal, null, authorities);
                auth.setDetails(new WebAuthenticationDetailsSource().buildDetails(request));
                SecurityContextHolder.getContext().setAuthentication(auth);
            }
        } catch (JwtException | IllegalArgumentException ex) {
            // Invalid/expired token: leave the context unauthenticated and let the
            // entry point return 401. Never leak the reason to the caller.
            SecurityContextHolder.clearContext();
        }

        chain.doFilter(request, response);
    }
}
