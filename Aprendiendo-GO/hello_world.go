package main

import (
	"fmt"
	"reflect"
)

func main() {

	// Hola mundo

	fmt.Println("Hello, World!")

	/*

		Comentario de bloque

	*/

	// Variables
	var myString string = "Hello, World!"
	fmt.Println(myString)

	myString = "Hello, World! 2"
	fmt.Println(myString)

	var myString2 = "Hello, World! 3" // Inferencia de tipo
	fmt.Println(myString2)

	var myInt int = 10
	fmt.Println(myInt)
	myInt = myInt + 10
	fmt.Println(myInt)
	fmt.Println(myInt + 10)
	fmt.Println(myInt)

	fmt.Println(myString, ", ", myInt)

	fmt.Println(reflect.TypeOf(myString))

	var myFloat float64 = 3.14
	fmt.Println(myFloat)
	fmt.Println(reflect.TypeOf(myFloat))

	fmt.Println(myInt + int(myFloat))
	fmt.Println(myFloat + float64(myInt))

	var myBool bool = true
	println(myBool)

	myString3 := "Hello, World! 4"
	fmt.Println(myString3)

	// Constantes
	const myConst string = "Esto es una constante"
	fmt.Println(myConst)

	// Control de flujo
	if myInt > 10 {
		fmt.Println("Mayor a 10")
	} else if myInt < 10 {
		fmt.Println("Menor a 10")
	} else {
		fmt.Println("Igual a 10")
	}

	// array

	var myArray [3]int
	myArray[0] = 1
	fmt.Println(myArray)

	// map

	myMap := make(map[string]int)
	myMap["one"] = 1
	myMap["two"] = 2
	fmt.Println(myMap)

	// list

	myList := []int{1, 2, 3}
	fmt.Println(myList)

	// list like pile

	myList = append(myList, 4)
	fmt.Println(myList)

	// for

	for i := 0; i < 10; i++ {
		fmt.Println(i)
	}

	for value := range myList {
		fmt.Println(value)
	}

	// funciones
	myFunction()

	// estructuras

	type myStruct struct {
		myField string
		age     int
	}

	myStruct2 := myStruct{"Hello, World!", 10}
	fmt.Println(myStruct2)

}

func myFunction() {
	fmt.Println("Mi función")
}
