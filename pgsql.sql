CREATE TABLE IF NOT EXISTS posts (
    num serial PRIMARY KEY,
    title varchar(255) NOT NULL,
    alt varchar(255) NOT NULL,
    image varchar(255) NOT NULL,
    posted timestamp with time zone NOT NULL,
    deleted boolean DEFAULT false NOT NULL
);

CREATE TABLE IF NOT EXISTS users (
    num serial PRIMARY KEY,
    name varchar(255) NOT NULL UNIQUE,
    email varchar(255) NOT NULL,
    password varchar(255),
    salt varchar(255),
    deleted boolean DEFAULT false NOT NULL
);

CREATE TABLE IF NOT EXISTS password_resets (
    num serial PRIMARY KEY,
    reset_token varchar(255) NOT NULL,
    salt varchar(255) NOT NULL,
    for_user serial references users(num),
    not_after timestamp with time zone NOT NULL,
    used boolean DEFAULT false NOT NULL
);
