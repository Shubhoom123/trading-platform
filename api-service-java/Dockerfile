# Multi-stage build for the Java API.
# Build context is the REPO ROOT (the protobuf plugin reads ../proto):
#   docker build -f api-service-java/Dockerfile -t trading/api-service .
FROM maven:3.9-eclipse-temurin-21 AS build

WORKDIR /src
COPY proto/ proto/
COPY api-service-java/ api-service-java/

WORKDIR /src/api-service-java
# Package the Spring Boot fat jar. Tests (Testcontainers) need Docker, so they
# run in CI, not in the image build.
RUN mvn -B -DskipTests clean package

# JRE-only runtime (no Maven, no JDK) keeps the final image small.
FROM eclipse-temurin:21-jre
WORKDIR /app
COPY --from=build /src/api-service-java/target/api-service-*.jar /app/app.jar
EXPOSE 8080
ENTRYPOINT ["java", "-jar", "/app/app.jar"]
