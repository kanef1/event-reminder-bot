CREATE TABLE events (
                        "eventId" SERIAL PRIMARY KEY,
                        "userTgId" BIGINT NOT NULL,
                        "message" TEXT NOT NULL,
                        "sendAt" TIMESTAMPTZ NOT NULL,
                        "createdAt" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_events_user ON events("userTgId");
CREATE INDEX idx_events_sendat ON events("sendAt");



