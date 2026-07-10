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
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.mockito.ArgumentCaptor;

import java.math.BigDecimal;
import java.util.Optional;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

// Unit test of the order-intake logic with mocked persistence + engine. No
// Spring, DB, or Kafka — exercises tick conversion, balance checks, and the
// async publish path directly.
class OrderServiceTest {

    private OrderRepository orders;
    private AccountRepository accounts;
    private MatchingEngineClient engine;
    private OrderService service;

    @BeforeEach
    void setup() {
        orders = mock(OrderRepository.class);
        accounts = mock(AccountRepository.class);
        engine = mock(MatchingEngineClient.class);
        service = new OrderService(orders, accounts, engine);
        // save() returns the entity it was given.
        when(orders.save(any(OrderEntity.class))).thenAnswer(inv -> inv.getArgument(0));
    }

    private void accountWithBalance(long balanceTicks) {
        when(accounts.findByUserId(1L)).thenReturn(Optional.of(new Account(1L, balanceTicks)));
    }

    @Test
    void limitSellConvertsPriceToTicksAndPublishesAsNew() {
        accountWithBalance(0); // sells don't need balance
        var req = new PlaceOrderRequest("AAPL", Side.SELL, OrderType.LIMIT,
                new BigDecimal("150.25"), 5L);

        OrderResponse resp = service.place(1L, req);

        assertEquals(15025L, resp.priceTicks(), "decimal -> integer ticks");
        assertEquals("NEW", resp.status(), "order accepted for async matching");
        verify(engine).submit(any(OrderEntity.class));
    }

    @Test
    void buyWithinBalanceReservesNotional() {
        accountWithBalance(100_000);
        var req = new PlaceOrderRequest("AAPL", Side.BUY, OrderType.LIMIT,
                new BigDecimal("150.25"), 5L); // notional = 15025 * 5 = 75125

        OrderResponse resp = service.place(1L, req);

        assertEquals("NEW", resp.status());
        verify(engine).submit(any(OrderEntity.class));
    }

    @Test
    void buyExceedingBalanceIsRejectedAndNotPublished() {
        accountWithBalance(1_000); // far below the notional
        var req = new PlaceOrderRequest("AAPL", Side.BUY, OrderType.LIMIT,
                new BigDecimal("150.25"), 5L);

        assertThrows(ApiException.class, () -> service.place(1L, req));
        verify(engine, never()).submit(any());
    }

    @Test
    void limitPriceFinerThanTickSizeIsRejected() {
        accountWithBalance(100_000);
        var req = new PlaceOrderRequest("AAPL", Side.SELL, OrderType.LIMIT,
                new BigDecimal("150.255"), 1L); // 3 decimal places > tick scale 2

        assertThrows(ApiException.class, () -> service.place(1L, req));
    }

    @Test
    void marketBuyIsRejectedUntilPriceReferenceExists() {
        accountWithBalance(100_000);
        var req = new PlaceOrderRequest("AAPL", Side.BUY, OrderType.MARKET, null, 5L);

        assertThrows(ApiException.class, () -> service.place(1L, req));
        verify(engine, never()).submit(any());
    }

    @Test
    void orderIsPersistedBeforeItIsPublished() {
        accountWithBalance(0);
        var req = new PlaceOrderRequest("MSFT", Side.SELL, OrderType.LIMIT,
                new BigDecimal("10.00"), 2L);

        service.place(1L, req);

        ArgumentCaptor<OrderEntity> saved = ArgumentCaptor.forClass(OrderEntity.class);
        verify(orders).save(saved.capture());
        assertEquals("MSFT", saved.getValue().getSymbol());
        assertEquals(1000L, saved.getValue().getPriceTicks());
    }
}
