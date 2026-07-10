package com.tradingplatform.api.repository;

import com.tradingplatform.api.domain.FillRecord;
import org.springframework.data.jpa.repository.JpaRepository;

public interface FillRepository extends JpaRepository<FillRecord, Long> {
    boolean existsBySequence(long sequence);
}
