To connect to a PostgreSQL database using the `psql` command with a connection string stored in the `PG_URL` environment variable, you can use the following command:

```bash
psql $PG_URL
```

Here’s a breakdown of how this works:

- **`psql`**: This is the command-line interface for PostgreSQL, used to connect to and interact with the PostgreSQL database.
- **`$PG_URL`**: This represents the environment variable `PG_URL` that presumably contains the connection string. The connection string usually includes the database host, port, user, password, and database name, formatted like so:
  ```
  postgresql://user:password@host:port/dbname
  ```
  
Make sure that the `PG_URL` variable is correctly set in your environment. You can check this by running:

```bash
echo $PG_URL
```

If you see the connection string, then you are all set to use it with `psql`. 

**Note**: If your connection string contains special characters or spaces, you might need to quote it.

In case the `psql` command is not found, or you're unsure if it's installed, you can verify its existence by running:

```bash
which psql
```

If you have any more specific needs or encounter any issues while connecting, please let me know!
