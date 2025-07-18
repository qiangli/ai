package db

import (
	"database/sql"
	"fmt"
	"net"
	"strings"

	"github.com/jackc/pgx/v5/pgproto3"
	_ "github.com/lib/pq"
	"github.com/xwb1989/sqlparser"

	"github.com/qiangli/ai/internal/log"
)

type PGLite struct {
	backend *pgproto3.Backend
	conn    net.Conn
	dbname  string

	db                 *sql.DB
	preparedStatements map[string]string
	activePortals      map[string]*activePortal
}

type activePortal struct {
	statementName string
	params        []interface{}
}

func NewPGLite(conn net.Conn, dbname string) *PGLite {
	backend := pgproto3.NewBackend(conn, conn)
	connHandler := &PGLite{
		backend:            backend,
		conn:               conn,
		dbname:             dbname,
		preparedStatements: make(map[string]string),
		activePortals:      make(map[string]*activePortal),
	}

	return connHandler
}

func (p *PGLite) Run() error {
	defer p.Close()

	err := p.handleStartup()
	if err != nil {
		return err
	}

	for {
		msg, err := p.backend.Receive()
		if err != nil {
			return fmt.Errorf("error receiving message: %w", err)
		}

		log.Debugf("Received: %T %#v\n", msg, msg)

		switch qmsg := msg.(type) {
		case *pgproto3.Query:
			query := qmsg.String

			log.Debugf("process query: %s\n", query)

			p.handleQuery(query)
			continue
		case *pgproto3.Terminate:
			if p.db != nil {
				p.db.Close()
				p.db = nil
			}
			log.Debugf("Client terminated connection\n")
			return nil
		case *pgproto3.Parse:
			name := qmsg.Name
			query := qmsg.Query

			log.Debugf("process Parse: %v, Name: %s\n", query, name)

			p.preparedStatements[name] = query
			p.send(&pgproto3.ParseComplete{})
			p.flush()
			continue
		case *pgproto3.Bind:
			portalName := qmsg.DestinationPortal
			statementName := qmsg.PreparedStatement

			log.Debugf("process Bind: %v, Name: %s\n", portalName, statementName)

			// Check if the prepared statement exists
			if _, exists := p.preparedStatements[statementName]; !exists {
				p.sendError(fmt.Errorf("prepared statement not found: %s", statementName))
				continue
			}

			// Convert parameter values
			paramValues := make([]interface{}, len(qmsg.Parameters))
			for i, paramVal := range qmsg.Parameters {
				paramValues[i] = string(paramVal)
			}

			// Store the portal information
			p.activePortals[portalName] = &activePortal{statementName: statementName, params: paramValues}

			p.send(&pgproto3.BindComplete{})
			p.flush()

		case *pgproto3.Execute:
			portalName := qmsg.Portal
			portal, exists := p.activePortals[portalName]

			log.Debugf("process Execute: %v, Name: %s exists: %v \n", portalName, portal, exists)

			if !exists {
				p.sendError(fmt.Errorf("portal not found: %s", portalName))
				continue
			}

			query := p.preparedStatements[portal.statementName]

			p.executeQuery(query, portal.params)

			// Optionally, you can remove the portal once executed if it's single-use
			// delete(p.activePortals, portalName)
		case *pgproto3.Describe:
			name := qmsg.Name
			objectType := qmsg.ObjectType

			fmt.Printf("process Describe: object name %v, type %v\n", name, objectType)

			switch objectType {
			case 'S': // Statement
				if query, exists := p.preparedStatements[name]; exists {
					// Determine fields based on the actual query analysis
					fields, err := getQueryMetadata(p.db, query)
					if err != nil {
						p.sendError(err)
					} else {
						p.send(&pgproto3.RowDescription{Fields: fields})
						continue
					}
				} else {
					p.sendError(fmt.Errorf("prepared statement not found: %s", name))
				}
			case 'P': // Portal
				// For simplicity, return no data as we're not managing portals here
				p.send(&pgproto3.NoData{})
			}

			p.send(&pgproto3.CommandComplete{CommandTag: []byte("DESCRIBE")})
			p.flush()
		case *pgproto3.Sync:
			log.Debugf("process Sync\n")

			// Send a ReadyForQuery response, indicating processing is complete
			p.send(&pgproto3.ReadyForQuery{TxStatus: 'I'})
			p.flush()
		case *pgproto3.Close:
			name := qmsg.Name
			objectType := qmsg.ObjectType

			log.Debugf("process Close: name: %s object type %c\n", name, objectType)

			// Perform the close operation based on the type
			switch objectType {
			case 'S': // Statement
				if _, exists := p.preparedStatements[name]; exists {
					delete(p.preparedStatements, name)
					log.Debugf("Closed statement: %s\n", name)
				}
			case 'P': // Portal
				if _, exists := p.activePortals[name]; exists {
					delete(p.activePortals, name)
					log.Debugf("Closed portal: %s\n", name)
				}
			}

			// Send CloseComplete message
			p.send(&pgproto3.CloseComplete{})
			p.flush()
		case *pgproto3.Flush:
			// Handle Flush message
			log.Debugf("process                                                                Flush\n")

			p.flush()
		default:
			log.Debugf("Not supported: %#v\n", msg)

			return fmt.Errorf("received message other than Query from client: %#v", msg)
		}
	}
}

// Helper function to retrieve query metadata
func getQueryMetadata(db *sql.DB, query string) ([]pgproto3.FieldDescription, error) {
	// This function should prepare a statement, execute an empty query to get metadata, and return FieldDescriptions
	stmt, err := db.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	fields := make([]pgproto3.FieldDescription, len(cols))
	for i, col := range cols {
		fields[i] = pgproto3.FieldDescription{
			Name:        []byte(col.Name()),
			DataTypeOID: 25, // Simplified example; derive OID from actual type
			// ... more attributes if needed
		}
	}
	return fields, nil
}

func (p *PGLite) handleStartup() error {
	// &{196608 map[client_encoding:UTF8 database:postgres user:postgres]}
	// &{196608 map[application_name:psql client_encoding:SQL_ASCII database:postgres user:postgres]}
	// &{196608 map[client_encoding:UTF8 database:postgres user:postgres]}
	msg, err := p.backend.ReceiveStartupMessage()
	if err != nil {
		return fmt.Errorf("error receiving startup message: %w", err)
	}

	log.Debugf("Received startup message %v\n", msg)

	auth := func() (*pgproto3.PasswordMessage, error) {
		// Request password from client
		authReq := &pgproto3.AuthenticationCleartextPassword{}
		buf := mustEncode(authReq.Encode(nil))
		_, err = p.conn.Write(buf)
		if err != nil {
			return nil, fmt.Errorf("error sending password request: %w", err)
		}
		p.flush()
		// Receive password
		pwdMsg, err := p.backend.Receive()
		if err != nil {
			return nil, fmt.Errorf("error receiving password message: %w", err)
		}
		pwd, ok := pwdMsg.(*pgproto3.PasswordMessage)
		if !ok {
			return nil, fmt.Errorf("expected PasswordMessage, got: %#v", pwdMsg)
		}
		return pwd, nil
	}

	// TODO
	// validate password
	validate := func(pwd string) bool {
		return pwd == "password"
	}

	switch smsg := msg.(type) {
	case *pgproto3.StartupMessage:
		user := smsg.Parameters["user"]
		dbname := smsg.Parameters["database"]
		if dbname == "" {
			dbname = p.dbname
		}
		// "pgAdmin 4 - DB:postgres"
		// "psql"
		appname := smsg.Parameters["application_name"]
		host := smsg.Parameters["host"]
		if host == "" {
			host = "localhost"
		}
		password := smsg.Parameters["password"]
		if password == "" {
			pwd, err := auth()
			if err != nil {
				p.sendError(err)
				return err
			}

			if !validate(pwd.Password) {
				p.sendFatal(fmt.Errorf("password authentication failed for user %q", user))
				return fmt.Errorf("authentication failed: invalid password")
			}

			password = pwd.Password
		}

		//
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable", host, user, password, dbname)
		log.Debugf("startup message %s from %s\n", dsn, appname)
		db, err := sql.Open("postgres", dsn)
		if err != nil {
			return err
		}
		p.db = db

		buf := mustEncode((&pgproto3.AuthenticationOk{}).Encode(nil))
		buf = mustEncode((&pgproto3.ReadyForQuery{TxStatus: 'I'}).Encode(buf))
		_, err = p.conn.Write(buf)
		if err != nil {
			return fmt.Errorf("error sending ready for query: %w", err)
		}
	case *pgproto3.SSLRequest:
		_, err = p.conn.Write([]byte("N"))
		if err != nil {
			return fmt.Errorf("error sending deny SSL request: %w", err)
		}
		return p.handleStartup()
	case *pgproto3.GSSEncRequest:
		log.Debugln("Received GSSEncRequest message")
		// Respond with 'N' to indicate that GSS encryption is not supported
		_, err = p.conn.Write([]byte("N"))
		if err != nil {
			return fmt.Errorf("error sending deny GSSEncRequest: %w", err)
		}
		return p.handleStartup()
	case *pgproto3.CancelRequest:
		log.Debugln("Received CancelRequest message")
		return p.Close()
	default:
		return fmt.Errorf("unknown startup message: %v", smsg)
	}

	return nil
}

func (p *PGLite) Close() error {
	if p.conn != nil {
		err := p.conn.Close()
		p.conn = nil
		return err
	}
	return nil
}

func mustEncode(buf []byte, err error) []byte {
	if err != nil {
		panic(err)
	}
	return buf
}

func (p *PGLite) handleQuery(query string) {
	query = strings.TrimSpace(query)
	log.Debugf("handleQuery: %q\n", query)

	stmt, err := sqlparser.Parse(query)
	if err != nil {
		log.Errorf("error parsing query: %s\n", query)
		// parser fails to parse some types of query.
		// best effort
		_, err := p.db.Exec(query)
		if err != nil {
			p.sendError(err)
			return
		}
		resp := commandTag(query)

		p.send(&pgproto3.CommandComplete{CommandTag: []byte(resp)})
		p.send(&pgproto3.ReadyForQuery{TxStatus: 'I'})
		p.flush()
		return
	}

	switch stmt.(type) {
	case *sqlparser.Select:
		p.executeQuery(query, nil)
	case *sqlparser.DDL:
		// DDL represents a CREATE, ALTER, DROP, RENAME or TRUNCATE statement.
		_, err := p.db.Exec(query)
		if err != nil {
			p.sendError(err)
			return
		}
		resp := commandTag(query)
		p.send(&pgproto3.CommandComplete{CommandTag: []byte(resp)})
	case *sqlparser.DBDDL:
		// DBDDL represents a CREATE, DROP database statement.
		p.sendError(fmt.Errorf("not supported. Query: %s", query))
		return
	default:
		// For non-select queries, just execute
		// insert/update/delete
		result, err := p.db.Exec(query)
		if err != nil {
			p.sendError(err)
			return
		}
		affected, _ := result.RowsAffected()
		var resp string

		// Determine the command tag based on operation
		switch stmt.(type) {
		case *sqlparser.Insert:
			resp = fmt.Sprintf("INSERT 0 %v", affected)
		case *sqlparser.Update:
			resp = fmt.Sprintf("UPDATE %v", affected)
		case *sqlparser.Delete:
			resp = fmt.Sprintf("DELETE %v", affected)
		case *sqlparser.Set:
			resp = fmt.Sprintf("%v rows affected", affected)
		case *sqlparser.Begin:
			resp = "BEGIN"
		case *sqlparser.Rollback:
			resp = "ROLLBACK"
		case *sqlparser.Commit:
			resp = "COMMIT"
		default:
			// ?
			resp = "COMMAND"
		}
		p.send(&pgproto3.CommandComplete{CommandTag: []byte(resp)})
	}

	p.send(&pgproto3.ReadyForQuery{TxStatus: 'I'})
	p.flush()
}

func (p *PGLite) executeQuery(query string, params []any) {
	log.Debugf("executeQuery: %q\n", query)

	stmt, err := p.db.Prepare(query)
	if err != nil {
		p.sendError(fmt.Errorf("error prepariing query: %s", err))
		return
	}
	defer stmt.Close()
	//
	var rows *sql.Rows

	if len(params) > 0 {
		rows, err = stmt.Query(params)
	} else {
		rows, err = stmt.Query()
	}
	if err != nil {
		p.sendError(err)
		return
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		p.sendError(err)
		return
	}
	// Send RowDescription
	fields := make([]pgproto3.FieldDescription, len(cols))
	for i, col := range cols {
		fields[i] = pgproto3.FieldDescription{
			Name:                 []byte(col),
			TableOID:             0,
			TableAttributeNumber: 0,
			DataTypeOID:          25, // TEXT oid
			DataTypeSize:         -1,
			TypeModifier:         -1,
			Format:               0, // text format
		}
	}
	p.send(&pgproto3.RowDescription{Fields: fields})

	rowCount := 0
	for rows.Next() {
		values, err := rowsValues(rows, len(cols))
		if err != nil {
			p.sendError(err)
			return
		}
		p.send(&pgproto3.DataRow{Values: values})
		rowCount++
	}
	if err := rows.Err(); err != nil {
		p.sendError(err)
		return
	}

	p.send(&pgproto3.CommandComplete{CommandTag: []byte(fmt.Sprintf("SELECT %d", rowCount))})
}

func rowsValues(rows *sql.Rows, numCols int) ([][]byte, error) {
	dest := make([]interface{}, numCols)
	for i := range dest {
		dest[i] = new(interface{})
	}

	if err := rows.Scan(dest...); err != nil {
		return nil, err
	}

	result := make([][]byte, numCols)
	for i, val := range dest {
		if b, ok := (*(val.(*interface{}))).([]byte); ok {
			result[i] = b
		} else {
			result[i] = []byte(fmt.Sprintf("%v", *(val.(*interface{}))))
		}

		log.Debugf("row[%v]: %s\n", i, string(result[i]))
	}

	return result, nil
}

func (p *PGLite) flush() {
	log.Debugln("flush")

	p.backend.Flush()
}

func (p *PGLite) send(msg pgproto3.BackendMessage) {
	log.Debugf("send: %#v\n", msg)

	p.backend.Send(msg)
}

func (p *PGLite) sendError(err error) {
	log.Debugf("sendError: %v\n", err)

	p.backend.Send(&pgproto3.ErrorResponse{
		Severity: "ERROR",
		Message:  err.Error(),
	})
	p.backend.Send(&pgproto3.ReadyForQuery{TxStatus: 'I'})
	p.backend.Flush()
}

func (p *PGLite) sendFatal(err error) {
	log.Debugf("sendFatal: %v\n", err)

	p.backend.Send(&pgproto3.ErrorResponse{
		Severity: "FATAL",
		Message:  err.Error(),
	})
	p.backend.Send(&pgproto3.ReadyForQuery{TxStatus: 'I'})
	p.backend.Flush()
}

// mainly for DDL
func commandTag(query string) string {
	q := strings.ToUpper(query)

	log.Debugf("commandTag: %s", q)

	var tag string

	fields := strings.Fields(q)
	switch len(fields) {
	case 0:
		tag = ""
	case 1:
		tag = fields[0]
	default:
		switch fields[0] {
		case "CREATE", "ALTER", "DROP":
			tag = fmt.Sprintf("%s %s", fields[0], fields[1])
		case "RENAME", "TRUNCATE":
			tag = fmt.Sprintf("%s %s", fields[0], fields[1])
		case "DO":
			tag = "DO"
		default:
			tag = fields[0]
		}
	}

	log.Debugf("tag created: %q\n", tag)

	return tag
}

func StartPG(address string, dbname string) {
	// conn, err := sql.Open("file::memory:?mode=memory&cache=shared")
	// db, err := sql.Open("sqlite", "test.db")
	// db, err := sql.Open("postgres", "host=localhost user=postgres password=password dbname=postgres sslmode=disable")
	// if err != nil {
	// 	log.Fatalf("failed to open database: %v", err)
	// }

	// defer db.Close()

	ln, err := net.Listen("tcp", address)
	if err != nil {
		log.Errorf("failed to listen on %s: %v\n", address, err)
		return
	}
	log.Infof("Postgres listening on %s...\n", address)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Debugf("failed to accept connection: %v\n", err)
			continue
		}

		log.Debugf("Accepted connection from: %v\n", conn.RemoteAddr())

		pg := NewPGLite(conn, dbname)
		go func() {
			err := pg.Run()
			if err != nil {
				log.Debugln(err)
			}
			log.Debugf("Closed connection from %v\n", conn.RemoteAddr())
		}()
	}
}
