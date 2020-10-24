package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/lib/pq"
	sqltrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/database/sql"
)

const (
	dbDriver = "pq"
)

var (
	host     string
	port     int
	user     string
	password string
	dbname   string
	instance Datasource
)

func init() {
	host = os.Getenv("PGHOST")
	port, _ = strconv.Atoi(os.Getenv("PGPORT"))
	user = os.Getenv("PGUSER")
	password = os.Getenv("PGPASSWORD")
	dbname = os.Getenv("PGDATABASE")
	instance = GetDatasourceInstance()
}

func main() {

	http.HandleFunc("/horarios", func(w http.ResponseWriter, r *http.Request) {

		cep, _ := strconv.ParseInt(r.URL.Query().Get("cep"), 10, 64)

		data, err := GetHorarios(r.Context(), cep)

		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
			return
		}

		if len(data) == 0 {
			w.WriteHeader(204)
			return
		}

		w.WriteHeader(200)
		w.Write(data)
	})

	http.HandleFunc("/cadastrar", func(w http.ResponseWriter, r *http.Request) {

		var err error
		var coletas []Coleta
		var ok bool

		body, err := ioutil.ReadAll(r.Body)

		err = json.Unmarshal(body, &coletas)

		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
			return
		}

		if ok, err = CadatrarColetas(coletas); ok {
			w.WriteHeader(200)
			w.Write([]byte("Cadastro realizado com sucesso"))
			return
		}

		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
			return
		}

	})

	log.Fatal(http.ListenAndServe(":5000", nil))

}

//CadatrarColetas create new colector route
func CadatrarColetas(coletas []Coleta) (result bool, err error) {
	var id int

	for _, coleta := range coletas {
		sqlStatement := `
		INSERT INTO coleta (cep, endereco, horario, dia)
			 VALUES ($1, $2, $3, $4)
			 RETURNING ID`

		err = instance.GetDB().QueryRow(sqlStatement, coleta.Cep, coleta.Endereco, coleta.Horario, coleta.Dia).Scan(&id)

		if err != nil {
			return false, err
		}

	}
	return true, err

}

//GetHorarios returns Cidade data with bbox to set focus on map
func GetHorarios(ctx context.Context, cep int64) ([]byte, error) {
	var data []byte
	err := instance.GetDB().QueryRowContext(ctx, `select jsonb_agg(t)
										   			from ( select id
													            , cep 
													            , endereco
													            , horario
													            , dia 
													         from coleta 
													         where cep = $1
													         order by dia, horario
												    ) t `, cep).Scan(&data)
	return data, err
}

//CreatePostgresDataSource cria uma nova instancia de PostgresDataSource
func CreatePostgresDataSource() *PostgresDataSource {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable search_path=sgpa_map,public", host, port, user, password, dbname)
	sqltrace.Register("pq", &pq.Driver{})
	db, err := sqltrace.Open(dbDriver, psqlInfo)
	if err != nil {
		panic(err.Error())
	}

	err = db.Ping()
	if err != nil {
		panic(err.Error())
	}

	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(time.Minute * 5)

	return &PostgresDataSource{db}
}

//Coleta Object from trash colector
type Coleta struct {
	Cep      int    `json:"cep"`
	Endereco string `json:"endereco"`
	Horario  string `json:"horario"`
	Dia      string `json:"dia"`
}

//GetDatasourceInstance singleton method to return the Datasource interface instance
func GetDatasourceInstance() Datasource {
	if instance == nil {
		instance = CreatePostgresDataSource()
	}
	return instance
}

//Datasource read only interface to access database connection
type Datasource interface {
	GetDB() *sql.DB
}

//PostgresDataSource instancia singleton que armazena uma referencia ao pool de conexao com o banco
type PostgresDataSource struct {
	db *sql.DB
}

//GetDB get instance of a connection to the database
func (p *PostgresDataSource) GetDB() *sql.DB {
	return p.db
}
