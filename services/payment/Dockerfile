FROM maven:3.9-eclipse-temurin-21 AS build
WORKDIR /src/services/payment
COPY services/payment/pom.xml ./
RUN mvn -B -q dependency:go-offline
COPY services/payment/src ./src
RUN mvn -B -q package -DskipTests

FROM eclipse-temurin:21-jre-alpine
RUN adduser -D -u 10001 app
COPY --from=build /src/services/payment/target/payment.jar /app/payment.jar
COPY contracts /app/contracts
ENV CONTRACTS_DIR=/app/contracts
USER app
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=5s --start-period=40s --retries=3 \
  CMD wget -qO- http://localhost:8080/health || exit 1
ENTRYPOINT ["java", "-XX:MaxRAMPercentage=75", "-jar", "/app/payment.jar"]
