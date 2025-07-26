--- 0000_init.sql ---

--
CREATE TABLE IF NOT EXISTS "User" (
	"id" uuid PRIMARY KEY NOT NULL,
    "created" timestamp NOT NULL,
	"name" varchar(256) NOT NULL,
	"data" json NOT NULL
);

--
CREATE TABLE IF NOT EXISTS "Chat" (
	"id" uuid PRIMARY KEY  NOT NULL,
    "userId" uuid NOT NULL,
	"created" timestamp NOT NULL
	"title" text,
	"data" json NOT NULL,
);

--
CREATE TABLE IF NOT EXISTS "Message" (
	"id" uuid PRIMARY KEY NOT NULL,
	"chatId" uuid NOT NULL,
	"created" timestamp NOT NULL
	"data" json NOT NULL,
);
