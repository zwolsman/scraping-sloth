CREATE TABLE prices
(
    gtin  BIGINT,
    title TEXT                      NOT NULL,
    date  DATE DEFAULT CURRENT_DATE NOT NULL,
    shop  TEXT                      NOT NULL,
    price DECIMAL(10, 2)            NOT NULL,

    PRIMARY KEY (gtin, date, shop)
);
