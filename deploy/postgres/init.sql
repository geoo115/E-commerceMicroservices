-- Database-per-service: every service gets its own database owned by its own
-- role, so a service cannot read or write another service's tables.
-- Passwords equal role names: this file is for local development only.

CREATE ROLE auth_svc      LOGIN PASSWORD 'auth_svc';
CREATE ROLE product_svc   LOGIN PASSWORD 'product_svc';
CREATE ROLE inventory_svc LOGIN PASSWORD 'inventory_svc';
CREATE ROLE order_svc     LOGIN PASSWORD 'order_svc';
CREATE ROLE payment_svc   LOGIN PASSWORD 'payment_svc';
CREATE ROLE cart_svc      LOGIN PASSWORD 'cart_svc';
CREATE ROLE review_svc    LOGIN PASSWORD 'review_svc';

CREATE DATABASE auth      OWNER auth_svc;
CREATE DATABASE products  OWNER product_svc;
CREATE DATABASE inventory OWNER inventory_svc;
CREATE DATABASE orders    OWNER order_svc;
CREATE DATABASE payments  OWNER payment_svc;
CREATE DATABASE carts     OWNER cart_svc;
CREATE DATABASE reviews   OWNER review_svc;

REVOKE CONNECT ON DATABASE auth, products, inventory, orders, payments, carts, reviews FROM PUBLIC;

-- Used by integration tests (TEST_DATABASE_URL).
CREATE DATABASE ecommerce_test;
