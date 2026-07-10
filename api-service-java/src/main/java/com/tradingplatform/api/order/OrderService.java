package com.tradingplatform.api.order;

import com.tradingplatform.api.common.ApiException;
import com.tradingplatform.api.domain.Account;
import com.tradingplatform.api.domain.OrderEntity;
import com.tradingplatform.api.domain.OrderType;
import com.tradingplatform.api.domain.Side;
import com.tradingplatform.api.engine.MatchingEngineClient;
import com.tradingplatform.api.order.dto.OrderResponse;
import com.tradingplatform.api.order.dto.PlaceOrderRequest;
import com.tradingplatform.api.repository.AccountRepository;
import com.tradingplatform.api.repository.OrderRepository;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.math.BigDecimal;
import java.util.List;

@Service
public class OrderService {

    // One tick = 0.01 of the quote currency. The engine only ever sees integer
    // ticks; decimals live at this boundary and nowhere below it.
    private static final int TICK_SCALE = 2;

    private final OrderRepository orders;
    private final AccountRepository accounts;
    private final MatchingEngineClient engine;

    public OrderService(OrderRepository orders, AccountRepository accounts,
                        MatchingEngineClient engine) {
        this.orders = orders;
        this.accounts = accounts;
        this.engine = engine;
    }

    @Transactional
    public OrderResponse place(Long userId, PlaceOrderRequest request) {
        Account account = accounts.findByUserId(userId)
                .orElseThrow(() -> ApiException.notFound("account not found"));

        long priceTicks = resolvePriceTicks(request);

        // Reserve the full buy notional up front, before the order leaves for the
        // engine. Because matching is now asynchronous (Kafka), we can't debit on
        // the fill round-trip; reserving at intake keeps the balance honest.
        // Price improvement (filling below the limit) is refunded later — a noted
        // Phase 5 simplification. Sells don't touch cash (no position ledger yet).
        if (request.side() == Side.BUY) {
            long notional = Math.multiplyExact(priceTicksForCheck(request, priceTicks),
                    request.quantity());
            if (notional > account.getBalanceTicks()) {
                throw ApiException.badRequest("insufficient balance for order notional");
            }
            account.debit(notional);
        }

        // Persist as NEW first so the DB-assigned id is what we publish; fills
        // reference this id when they come back on the fills topic.
        OrderEntity order = orders.save(new OrderEntity(
                account.getId(), request.symbol(), request.side(), request.type(),
                priceTicks, request.quantity()));

        // Publish to Kafka. The order is durable (NEW) regardless of match timing;
        // OrderFillConsumer advances it to PARTIALLY_FILLED/FILLED asynchronously.
        engine.submit(order);

        return OrderResponse.from(order);
    }

    @Transactional(readOnly = true)
    public List<OrderResponse> history(Long userId) {
        Account account = accounts.findByUserId(userId)
                .orElseThrow(() -> ApiException.notFound("account not found"));
        return orders.findByAccountIdOrderByCreatedAtDesc(account.getId()).stream()
                .map(OrderResponse::from)
                .toList();
    }

    private long resolvePriceTicks(PlaceOrderRequest request) {
        if (request.type() == OrderType.MARKET) {
            return 0L; // engine ignores price for market orders
        }
        if (request.price() == null || request.price().signum() <= 0) {
            throw ApiException.badRequest("limit orders require a positive price");
        }
        if (request.price().scale() > TICK_SCALE) {
            throw ApiException.badRequest("price precision exceeds tick size (0.01)");
        }
        return request.price().movePointRight(TICK_SCALE).longValueExact();
    }

    // For market buys we have no limit price to size the reservation against.
    // A production system reserves against a worst-case/last price; here we
    // require limit orders for the balance path and treat market price as 0.
    private long priceTicksForCheck(PlaceOrderRequest request, long priceTicks) {
        if (request.type() == OrderType.MARKET) {
            throw ApiException.badRequest(
                    "market buys are not supported until a price reference exists (Phase 5)");
        }
        return priceTicks;
    }

    // Kept for symmetry / documentation of the tick model.
    static BigDecimal ticksToDecimal(long ticks) {
        return BigDecimal.valueOf(ticks).movePointLeft(TICK_SCALE);
    }
}
