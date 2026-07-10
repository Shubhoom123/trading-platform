package com.tradingplatform.api.account;

import com.tradingplatform.api.common.ApiException;
import com.tradingplatform.api.domain.Account;
import com.tradingplatform.api.repository.AccountRepository;
import com.tradingplatform.api.security.AuthenticatedUser;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

// Exposes the authenticated user's account. The gateway proxies GET /api/account
// here so the frontend can show the available balance.
@RestController
@RequestMapping("/api/account")
public class AccountController {

    private final AccountRepository accounts;

    public AccountController(AccountRepository accounts) {
        this.accounts = accounts;
    }

    public record AccountResponse(Long accountId, long balanceTicks) {
    }

    @GetMapping
    public AccountResponse me(@AuthenticationPrincipal AuthenticatedUser user) {
        Account account = accounts.findByUserId(user.userId())
                .orElseThrow(() -> ApiException.notFound("account not found"));
        return new AccountResponse(account.getId(), account.getBalanceTicks());
    }
}
