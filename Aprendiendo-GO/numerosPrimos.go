package main

import (
	"fmt"
)

func main() {
	var numero int
	fmt.Print("Ingrese un número: ") // Se usa Print para no hacer un salto de línea
	fmt.Scan(&numero)
	if primo(numero) {
		fmt.Println("El número es primo")
	} else {
		fmt.Println("El número no es primo")
	}
}

func primo(n int) bool {
	if n <= 1 {
		return false
	}
	for i := 2; i*i <= n; i++ {
		if n%i == 0 {
			return false
		}
	}
	return true
}
