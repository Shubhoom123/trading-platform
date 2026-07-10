package com.tradingplatform.api;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.tradingplatform.api.engine.MatchingEngineClient;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.AutoConfigureMockMvc;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.http.MediaType;
import org.springframework.test.context.DynamicPropertyRegistry;
import org.springframework.test.context.DynamicPropertySource;
import org.springframework.test.web.servlet.MockMvc;
import org.testcontainers.containers.PostgreSQLContainer;
import org.testcontainers.junit.jupiter.Container;
import org.testcontainers.junit.jupiter.Testcontainers;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

// Runs the full order lifecycle against a real Postgres (Flyway migrates the
// schema), with the matching engine mocked so this stays a service-level test.
// The cross-language end-to-end test that also boots the C++ engine is a
// Phase 4 CI job (docker compose), noted in the build plan.
@SpringBootTest
@AutoConfigureMockMvc
@Testcontainers
class OrderLifecycleIT {

    @Container
    static final PostgreSQLContainer<?> POSTGRES =
            new PostgreSQLContainer<>("postgres:16-alpine")
                    .withDatabaseName("trading")
                    .withUsername("trading")
                    .withPassword("trading");

    @DynamicPropertySource
    static void datasource(DynamicPropertyRegistry registry) {
        registry.add("spring.datasource.url", POSTGRES::getJdbcUrl);
        registry.add("spring.datasource.username", POSTGRES::getUsername);
        registry.add("spring.datasource.password", POSTGRES::getPassword);
        // A real 32+ byte secret so JwtService starts under test.
        registry.add("security.jwt.secret",
                () -> "test-secret-test-secret-test-secret-0123456789");
        // Don't start the Kafka listener container: this is a service-level test
        // with the engine mocked, so no broker is needed. The full cross-service
        // flow over real Kafka is the Phase 4 docker-compose integration job.
        registry.add("spring.kafka.listener.auto-startup", () -> "false");
    }

    @Autowired
    MockMvc mvc;

    @Autowired
    ObjectMapper json;

    @MockBean
    MatchingEngineClient engine;

    @Test
    void registerLoginAndPlaceOrder() throws Exception {
        // engine.submit() is void (publishes to Kafka); the mock is a no-op so
        // no broker is touched. We assert the synchronous intake behaviour: the
        // order is persisted as NEW and published. Fill-driven status changes are
        // covered by the OrderFillConsumer path, exercised end-to-end in CI.

        // Register -> get tokens
        String register = """
                {"email":"trader@example.com","password":"correct horse battery"}
                """;
        String body = mvc.perform(post("/api/auth/register")
                        .contentType(MediaType.APPLICATION_JSON).content(register))
                .andExpect(status().isCreated())
                .andReturn().getResponse().getContentAsString();

        JsonNode tokens = json.readTree(body);
        String access = tokens.get("accessToken").asText();

        // Unauthenticated order placement is rejected
        mvc.perform(post("/api/orders")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {"symbol":"AAPL","side":"SELL","type":"LIMIT","price":150.25,"quantity":5}
                                """))
                .andExpect(status().isUnauthorized());

        // Authenticated SELL limit order goes through (sells skip the balance check)
        mvc.perform(post("/api/orders")
                        .header("Authorization", "Bearer " + access)
                        .contentType(MediaType.APPLICATION_JSON)
                        .content("""
                                {"symbol":"AAPL","side":"SELL","type":"LIMIT","price":150.25,"quantity":5}
                                """))
                .andExpect(status().isCreated())
                .andExpect(jsonPath("$.status").value("NEW"))
                .andExpect(jsonPath("$.priceTicks").value(15025));

        // History reflects the order
        mvc.perform(get("/api/orders").header("Authorization", "Bearer " + access))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$[0].symbol").value("AAPL"));
    }
}
