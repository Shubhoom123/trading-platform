package com.tradingplatform.api.order;

import com.tradingplatform.api.order.dto.OrderResponse;
import com.tradingplatform.api.order.dto.PlaceOrderRequest;
import com.tradingplatform.api.security.AuthenticatedUser;
import jakarta.validation.Valid;
import org.springframework.http.HttpStatus;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestController;

import java.util.List;

@RestController
@RequestMapping("/api/orders")
public class OrderController {

    private final OrderService orderService;

    public OrderController(OrderService orderService) {
        this.orderService = orderService;
    }

    @PostMapping
    @ResponseStatus(HttpStatus.CREATED)
    public OrderResponse place(@AuthenticationPrincipal AuthenticatedUser user,
                               @Valid @RequestBody PlaceOrderRequest request) {
        return orderService.place(user.userId(), request);
    }

    @GetMapping
    public List<OrderResponse> history(@AuthenticationPrincipal AuthenticatedUser user) {
        return orderService.history(user.userId());
    }
}
