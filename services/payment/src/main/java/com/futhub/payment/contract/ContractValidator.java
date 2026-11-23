package com.futhub.payment.contract;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.time.OffsetDateTime;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.UUID;
import java.util.stream.Collectors;
import java.util.stream.Stream;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.futhub.payment.config.PaymentProperties;
import com.networknt.schema.JsonSchema;
import com.networknt.schema.JsonSchemaFactory;
import com.networknt.schema.SchemaLocation;
import com.networknt.schema.SchemaValidatorsConfig;
import com.networknt.schema.SpecVersion;
import com.networknt.schema.ValidationMessage;
import org.springframework.stereotype.Component;

@Component
public class ContractValidator {

    private static final String EVENT_SCHEMA_SUFFIX = ".schema.json";
    private static final List<String> CANDIDATE_DIRECTORIES = List.of("contracts", "../../contracts", "../contracts");

    private final ObjectMapper mapper;
    private final Map<String, JsonSchema> schemasByType;

    public ContractValidator(PaymentProperties properties, ObjectMapper mapper) {
        this.mapper = mapper;
        this.schemasByType = loadSchemas(resolveDirectory(properties.contractsDir()));
    }

    public Envelope validate(byte[] body) {
        JsonNode tree = parse(body);
        String type = tree.path("type").asText("");

        JsonSchema schema = schemasByType.get(type);
        if (schema == null) {
            throw new InvalidMessageException("unsupported message type '" + type + "'");
        }

        Set<ValidationMessage> violations = schema.validate(tree);
        if (!violations.isEmpty()) {
            String details = violations.stream().map(ValidationMessage::getMessage).collect(Collectors.joining("; "));
            throw new InvalidMessageException("message does not match the " + type + " contract: " + details);
        }

        return new Envelope(
                UUID.fromString(tree.get("messageId").asText()),
                UUID.fromString(tree.get("correlationId").asText()),
                type,
                tree.get("version").asInt(),
                OffsetDateTime.parse(tree.get("occurredAt").asText()).toInstant(),
                tree.get("payload"));
    }

    private JsonNode parse(byte[] body) {
        try {
            JsonNode tree = mapper.readTree(body);
            if (tree == null || !tree.isObject()) {
                throw new InvalidMessageException("body is not a JSON object");
            }
            return tree;
        } catch (IOException e) {
            throw new InvalidMessageException("body is not valid JSON", e);
        }
    }

    private static Map<String, JsonSchema> loadSchemas(Path directory) {
        Path eventsDirectory = directory.resolve("events");
        JsonSchemaFactory factory = JsonSchemaFactory.getInstance(SpecVersion.VersionFlag.V202012);
        SchemaValidatorsConfig config = SchemaValidatorsConfig.builder().formatAssertionsEnabled(true).build();

        Map<String, JsonSchema> schemas = new HashMap<>();
        try (Stream<Path> files = Files.list(eventsDirectory)) {
            files.filter(file -> file.getFileName().toString().endsWith(EVENT_SCHEMA_SUFFIX)).forEach(file -> {
                String fileName = file.getFileName().toString();
                String type = fileName.substring(0, fileName.length() - EVENT_SCHEMA_SUFFIX.length());
                schemas.put(type, factory.getSchema(SchemaLocation.of(file.toUri().toString()), config));
            });
        } catch (IOException e) {
            throw new IllegalStateException("could not read event contracts from " + eventsDirectory, e);
        }
        if (schemas.isEmpty()) {
            throw new IllegalStateException("no event contracts found in " + eventsDirectory);
        }
        return schemas;
    }

    private static Path resolveDirectory(String configured) {
        List<String> candidates = configured == null || configured.isBlank() ? CANDIDATE_DIRECTORIES : List.of(configured);
        for (String candidate : candidates) {
            Path path = Path.of(candidate).toAbsolutePath().normalize();
            if (Files.isDirectory(path.resolve("events"))) {
                return path;
            }
        }
        throw new IllegalStateException("contracts directory not found, set CONTRACTS_DIR (tried " + candidates + ")");
    }
}
