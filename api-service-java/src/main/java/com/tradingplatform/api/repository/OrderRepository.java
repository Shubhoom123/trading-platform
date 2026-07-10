package com.tradingplatform.api.repository;

import com.tradingplatform.api.domain.OrderEntity;
import org.springframework.data.jpa.repository.JpaRepository;

import java.util.List;

public interface OrderRepository extends JpaRepository<OrderEntity, Long> {
    List<OrderEntity> findByAccountIdOrderByCreatedAtDesc(Long accountId);
}
