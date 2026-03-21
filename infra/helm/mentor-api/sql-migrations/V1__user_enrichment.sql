-- =============================================================================
-- V1: user-enrichment Flink SQL job
--
-- Joins two Debezium CDC streams from Kratos PostgreSQL:
--   identity.users.created        (public.identities)
--   identity.addresses.created    (public.identity_verifiable_addresses)
--
-- Produces domain event to mentor-api:
--   students.onboarding.ready     — emitted when a parent is created AND
--                                   their email address is verified
--
-- Routes invalid CREATE events to DLQ:
--   identity.users.created.dlq    — null_id, null_email, not_parent
--
-- Event time: uses __ts_ms (DB transaction time), not Kafka delivery timestamp.
-- This ensures correct Interval JOIN even when CDC delivery is delayed.
--
-- Variable substitution: ${VAR} is replaced at runtime by SqlRunner
-- using environment variables defined in the FlinkDeployment pod.
-- =============================================================================

-- ==== SOURCE 1: identities (Debezium ExtractNewRecordState flat payload) ====
CREATE TABLE users_created (
  `id`        STRING,
  `schema_id` STRING,
  `nid`       STRING,
  `traits`    STRING,
  `state`     STRING,
  `__op`      STRING,
  `__ts_ms`   BIGINT,
  event_time  AS TO_TIMESTAMP(FROM_UNIXTIME(`__ts_ms` / 1000)),
  WATERMARK FOR event_time AS event_time - INTERVAL '5' SECOND
) WITH (
  'connector'                          = 'kafka',
  'topic'                              = 'identity.users.created',
  'properties.bootstrap.servers'       = '${bootstrapServers}',
  'properties.security.protocol'       = 'SSL',
  'properties.ssl.truststore.location' = '${truststoreLocation}',
  'properties.ssl.truststore.password' = '${truststorePassword}',
  'properties.ssl.keystore.location'   = '${keystoreLocation}',
  'properties.ssl.keystore.password'   = '${keystorePassword}',
  'properties.group.id'                = 'flink-user-enrichment',
  'format'                             = 'avro-confluent',
  'avro-confluent.url'                 = '${schemaRegistryUrl}',
  'scan.startup.mode'                  = 'earliest-offset'
);

-- ==== SOURCE 2: identity_verifiable_addresses ====
CREATE TABLE addresses_created (
  `id`          STRING,
  `identity_id` STRING,
  `status`      STRING,
  `via`         STRING,
  `value`       STRING,
  `verified`    BOOLEAN,
  `__op`        STRING,
  `__ts_ms`     BIGINT,
  event_time    AS TO_TIMESTAMP(FROM_UNIXTIME(`__ts_ms` / 1000)),
  WATERMARK FOR event_time AS event_time - INTERVAL '5' SECOND
) WITH (
  'connector'                          = 'kafka',
  'topic'                              = 'identity.addresses.created',
  'properties.bootstrap.servers'       = '${bootstrapServers}',
  'properties.security.protocol'       = 'SSL',
  'properties.ssl.truststore.location' = '${truststoreLocation}',
  'properties.ssl.truststore.password' = '${truststorePassword}',
  'properties.ssl.keystore.location'   = '${keystoreLocation}',
  'properties.ssl.keystore.password'   = '${keystorePassword}',
  'properties.group.id'                = 'flink-user-enrichment',
  'format'                             = 'avro-confluent',
  'avro-confluent.url'                 = '${schemaRegistryUrl}',
  'scan.startup.mode'                  = 'earliest-offset'
);

-- ==== SINK: students.onboarding.ready ====
CREATE TABLE students_onboarding_ready (
  event_id    STRING,
  user_id     STRING,
  email       STRING,
  first_name  STRING,
  last_name   STRING,
  role        STRING,
  occurred_at STRING
) WITH (
  'connector'                          = 'kafka',
  'topic'                              = 'students.onboarding.ready',
  'properties.bootstrap.servers'       = '${bootstrapServers}',
  'properties.security.protocol'       = 'SSL',
  'properties.ssl.truststore.location' = '${truststoreLocation}',
  'properties.ssl.truststore.password' = '${truststorePassword}',
  'properties.ssl.keystore.location'   = '${keystoreLocation}',
  'properties.ssl.keystore.password'   = '${keystorePassword}',
  'format'                             = 'avro-confluent',
  'avro-confluent.url'                 = '${schemaRegistryUrl}'
);

-- ==== DLQ: invalid CREATE events ====
CREATE TABLE users_dlq (
  raw_id     STRING,
  raw_traits STRING,
  reason     STRING,
  ts_ms      BIGINT
) WITH (
  'connector'                          = 'kafka',
  'topic'                              = 'identity.users.created.dlq',
  'properties.bootstrap.servers'       = '${bootstrapServers}',
  'properties.security.protocol'       = 'SSL',
  'properties.ssl.truststore.location' = '${truststoreLocation}',
  'properties.ssl.truststore.password' = '${truststorePassword}',
  'properties.ssl.keystore.location'   = '${keystoreLocation}',
  'properties.ssl.keystore.password'   = '${keystorePassword}',
  'format'                             = 'json'
);

-- ==== INTERVAL JOIN: users × addresses ====
-- Stateful: Flink keeps state for both streams in RocksDB for up to 1 hour.
-- Emits when: parent identity created (__op='c') AND email verified.
-- Window ±1h covers both immediate and delayed email verification.
INSERT INTO students_onboarding_ready
SELECT
  CAST(UUID() AS STRING)                 AS event_id,
  u.`id`                                 AS user_id,
  JSON_VALUE(u.`traits`, '$.email')      AS email,
  JSON_VALUE(u.`traits`, '$.name.first') AS first_name,
  JSON_VALUE(u.`traits`, '$.name.last')  AS last_name,
  JSON_VALUE(u.`traits`, '$.role')       AS role,
  CAST(u.event_time AS STRING)           AS occurred_at
FROM users_created u
JOIN addresses_created a
  ON u.`id` = a.`identity_id`
  AND a.event_time BETWEEN u.event_time - INTERVAL '1' HOUR
                       AND u.event_time + INTERVAL '1' HOUR
WHERE u.`__op` = 'c'
  AND u.`id` IS NOT NULL
  AND JSON_VALUE(u.`traits`, '$.role') = 'parent'
  AND JSON_VALUE(u.`traits`, '$.email') IS NOT NULL
  AND a.`verified` = TRUE
  AND a.`via` = 'email';

-- ==== DLQ routing: invalid CREATE events ====
INSERT INTO users_dlq
SELECT
  u.`id`      AS raw_id,
  u.`traits`  AS raw_traits,
  CASE
    WHEN u.`id` IS NULL                                THEN 'null_id'
    WHEN JSON_VALUE(u.`traits`, '$.email') IS NULL     THEN 'null_email'
    WHEN JSON_VALUE(u.`traits`, '$.role') <> 'parent'  THEN 'not_parent'
    ELSE 'unknown'
  END AS reason,
  u.`__ts_ms` AS ts_ms
FROM users_created u
WHERE u.`__op` = 'c'
  AND (
    u.`id` IS NULL
    OR JSON_VALUE(u.`traits`, '$.email') IS NULL
    OR JSON_VALUE(u.`traits`, '$.role') <> 'parent'
  );
