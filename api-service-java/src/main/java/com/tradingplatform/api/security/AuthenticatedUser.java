package com.tradingplatform.api.security;

// The authenticated principal placed on the SecurityContext by the JWT filter.
// Controllers read it via @AuthenticationPrincipal.
public record AuthenticatedUser(Long userId, String email, String role) {
}
