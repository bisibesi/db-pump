-- 1. 기초 테이블 생성 (부모부터)
CREATE TABLE country (
    country_id NUMBER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    country VARCHAR2(50) NOT NULL,
    last_update TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE city (
    city_id NUMBER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    city VARCHAR2(50) NOT NULL,
    country_id NUMBER NOT NULL,
    last_update TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_city_country FOREIGN KEY (country_id) REFERENCES country(country_id)
);

CREATE TABLE address (
    address_id NUMBER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    address VARCHAR2(50) NOT NULL,
    address2 VARCHAR2(50),
    district VARCHAR2(20) NOT NULL,
    city_id NUMBER NOT NULL,
    postal_code VARCHAR2(10),
    phone VARCHAR2(20) NOT NULL,
    last_update TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_address_city FOREIGN KEY (city_id) REFERENCES city(city_id)
);

CREATE TABLE category (
    category_id NUMBER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name VARCHAR2(25) NOT NULL,
    last_update TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE language (
    language_id NUMBER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name CHAR(20) NOT NULL,
    last_update TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE film (
    film_id NUMBER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    title VARCHAR2(255) NOT NULL,
    description CLOB,
    release_year NUMBER(4),
    language_id NUMBER NOT NULL,
    rental_duration NUMBER DEFAULT 3 NOT NULL,
    rental_rate NUMBER(4,2) DEFAULT 4.99 NOT NULL,
    length NUMBER,
    replacement_cost NUMBER(5,2) DEFAULT 19.99 NOT NULL,
    rating VARCHAR2(10) DEFAULT 'G',
    last_update TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_film_language FOREIGN KEY (language_id) REFERENCES language(language_id)
);

CREATE TABLE actor (
    actor_id NUMBER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    first_name VARCHAR2(45) NOT NULL,
    last_name VARCHAR2(45) NOT NULL,
    last_update TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 2. 관계 테이블 및 자식 테이블 (의존성 순서 준수)
CREATE TABLE film_actor (
    actor_id NUMBER NOT NULL,
    film_id NUMBER NOT NULL,
    last_update TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (actor_id, film_id),
    CONSTRAINT fk_fa_actor FOREIGN KEY (actor_id) REFERENCES actor(actor_id),
    CONSTRAINT fk_fa_film FOREIGN KEY (film_id) REFERENCES film(film_id)
);

CREATE TABLE store (
    store_id NUMBER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    manager_staff_id NUMBER,
    address_id NUMBER NOT NULL,
    last_update TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_store_address FOREIGN KEY (address_id) REFERENCES address(address_id)
);

CREATE TABLE staff (
    staff_id NUMBER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    first_name VARCHAR2(45) NOT NULL,
    last_name VARCHAR2(45) NOT NULL,
    address_id NUMBER NOT NULL,
    email VARCHAR2(50),
    store_id NUMBER NOT NULL,
    active NUMBER(1) DEFAULT 1 NOT NULL,
    username VARCHAR2(16) NOT NULL,
    password VARCHAR2(40),
    last_update TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_staff_address FOREIGN KEY (address_id) REFERENCES address(address_id),
    CONSTRAINT fk_staff_store FOREIGN KEY (store_id) REFERENCES store(store_id)
);