package api // Nombre del paquete

import ( // Importar paquetes
	"encoding/json" // Importar el paquete json, nos ayuda a codificar y decodificar JSON
	"net/http"

	"github.com/google/uuid" // Importar el paquete uuid, nos ayuda a generar un id único
	"github.com/gorilla/mux" // Importar el paquete mux, nos ayuda a manejar rutas
)

type Item struct { // Definir una estructura
	Name  string    `json:"name"`  // Nombre del item
	Price int       `json:"price"` // Precio del item
	ID    uuid.UUID `json:"id"`    // ID del item
}

type Server struct { // Definir una estructura
	*mux.Router // Agregar un router

	hotweels []Item // Lista de items
}

func NewServer() *Server { // Define una función llamada NewServer que crea y retorna un puntero a un nuevo Server.
	s := &Server{
		// Crea una nueva instancia de la estructura Server y la asigna a la variable s.
		Router: mux.NewRouter(),
		// Inicializa el campo Router de la estructura Server con un nuevo enrutador (Router) de la librería mux.
		hotweels: []Item{},
		// Inicializa el campo hotwheels
	}

	s.routes()
	return s
}

func (s *Server) routes() { // Crear una función que se usa oara e usa para configurar las rutas HTTP que el servidor manejará.
	s.HandleFunc("/hotweels", s.createHotweelItem()).Methods(http.MethodPost)         // Crear un endpoint para agregar un item
	s.HandleFunc("/hotweels", s.listHotwheelItems()).Methods(http.MethodGet)          // Crear un endpoint para obtener la lista de items
	s.HandleFunc("/hotweels/{id}", s.removeHotwheelItem()).Methods(http.MethodDelete) // Crear un endpoint para eliminar un item
}

func (s *Server) createHotweelItem() http.HandlerFunc { // Esta función es un método de la estructura Server.
	// Retorna una función que maneja una petición HTTP (http.HandlerFunc).
	return func(w http.ResponseWriter, r *http.Request) { // La función interna es la que efectivamente manejará las peticiones HTTP.
		var i Item // Se declara una variable de tipo Item donde se almacenarán los datos recibidos en la petición.

		if err := json.NewDecoder(r.Body).Decode(&i); err != nil { // Se decodifica el cuerpo de la petición (r.Body) desde JSON a la estructura Item.
			// Si ocurre un error durante la decodificación, se maneja el error:
			http.Error(w, err.Error(), http.StatusBadRequest) // Se responde con un código de estado HTTP 400 (Bad Request) y el mensaje de error.
			return
			// Se finaliza la ejecución de la función en caso de error.
		}

		i.ID = uuid.New() // Se genera un nuevo UUID para el campo ID del item y se asigna a i.ID.

		s.hotweels = append(s.hotweels, i) // Se añade el nuevo item `i` al slice `shoppingItems` de la estructura Server.

		w.Header().Set("Content-Type", "application/json") // Se establece el encabezado de la respuesta HTTP, indicando que el contenido es de tipo JSON.

		if err := json.NewEncoder(w).Encode(i); err != nil { // Codifica el item `i` a JSON y se escribe en la respuesta `w`.
			// Si ocurre un error durante la codificación, se maneja el error:
			http.Error(w, err.Error(), http.StatusInternalServerError) // Responde con un código de estado HTTP 500 (Internal Server Error) y el mensaje de error.
			return
			// Finaliza la ejecución de la función en caso de error.
		}
	}
}

func (s *Server) listHotwheelItems() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Establece el encabezado de la respuesta HTTP para indicar que el contenido es JSON.

		if err := json.NewEncoder(w).Encode(s.hotweels); err != nil {
			// Codifica la lista de items (shoppingItems) a JSON y la escribe en la respuesta.

			http.Error(w, err.Error(), http.StatusInternalServerError)
			// Responde con un código de estado HTTP 500 (Internal Server Error) y el mensaje de error.
			return
		}
	}
}

func (s *Server) removeHotwheelItem() http.HandlerFunc { // Retorna una función que maneja una petición HTTP para eliminar un item por su ID.
	return func(w http.ResponseWriter, r *http.Request) {
		idStr, _ := mux.Vars(r)["id"] // Obtiene el ID del item desde las variables de la URL (path parameters).

		id, err := uuid.Parse(idStr) // Convierte el ID de string a UUID.

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			// Si falla, responde con un código HTTP 400 (Bad Request) y el mensaje de error.
			return
		}

		for i, item := range s.hotweels {
			// Recorre la lista de items para encontrar el que tiene el ID especificado.

			if item.ID == id {
				// Si encuentra el item con el ID coincidente:

				s.hotweels = append(s.hotweels[:i], s.hotweels[i+1:]...)
				// Elimina el item de la lista utilizando una técnica de slicing.
				break
			}
		}
	}
}
