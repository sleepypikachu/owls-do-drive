CREATE TABLE IF NOT EXISTS posts (
    num serial PRIMARY KEY,
    title varchar(255) NOT NULL,
    alt varchar(255) NOT NULL,
    image varchar(255) NOT NULL,
    posted timestamp with time zone NOT NULL,
    deleted boolean DEFAULT false NOT NULL
);

CREATE TABLE IF NOT EXISTS users (
    id serial PRIMARY KEY,
    name varchar(255) NOT NULL,
    password varchar(255) NOT NULL,
    deleted boolean DEFAULT false NOT NULL
);
