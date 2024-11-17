package main

import (
	"fmt"
	"math"
	"time"
)

// Константы симуляции
const (
	gravity            = 9.81     // Ускорение свободного падения (м/с^2)
	maxThrustPerEngine = 2000000  // Максимальная тяга одного двигателя (Н)
	emptyMass          = 120000   // Масса первой ступени без топлива (кг)
	initialFuelMass    = 300000   // Начальный запас топлива первой ступени (кг)
	secondStageMass    = 50000    // Масса второй ступени (кг)
	fuelBurnRate       = 250      // Расход топлива на один двигатель (кг/с)
	totalEngines       = 33       // Общее количество двигателей
	timeStep           = 0.1      // Шаг симуляции (секунды)
	orbitalHeight      = 80000    // Орбитальная высота цели (м)
	orbitalVelocity    = 27000    // Орбитальная скорость цели (м/с)
	beaconX            = 0        // X-координата башни-маяка
	beaconY            = 0        // Y-координата башни-маяка
	landingTolerance   = 0.1      // Допустимое расстояние до центра башни для успешной посадки (м)
	dragCoefficient    = 0.5      // Коэффициент лобового сопротивления
	crossSectionalArea = 10.0     // Лобовая площадь ракеты (м^2)
	airDensitySeaLevel = 1.225    // Плотность воздуха на уровне моря (кг/м^3)
)

// Rocket описывает состояние ракеты
type Rocket struct {
	Position       float64 // Положение в вертикальном направлении (м)
	Velocity       float64 // Скорость (м/с)
	Acceleration   float64 // Ускорение (м/с^2)
	Altitude       float64 // Высота (м)
	Mass           float64 // Текущая масса (кг)
	FuelMass       float64 // Текущий запас топлива (кг)
	ActiveEngines  int     // Количество активных двигателей
	SecondStage    bool    // Состояние второй ступени (true, если прикреплена)
	X, Y           float64 // Координаты для моделирования GPS
}

// CalculateAirDensity рассчитывает плотность воздуха в зависимости от высоты
func CalculateAirDensity(altitude float64) float64 {
	// Упрощенная модель: плотность экспоненциально уменьшается с высотой
	scaleHeight := 8500.0 // Средняя высота масштаба атмосферы (м)
	return airDensitySeaLevel * math.Exp(-altitude/scaleHeight)
}

// CalculateDragForce рассчитывает силу сопротивления воздуха
func (r *Rocket) CalculateDragForce() float64 {
	airDensity := CalculateAirDensity(r.Altitude)
	dragForce := 0.5 * dragCoefficient * airDensity * r.Velocity * r.Velocity * crossSectionalArea
	// Сопротивление воздуха всегда направлено против направления движения
	if r.Velocity > 0 {
		return -dragForce
	}
	return dragForce
}

// UpdateState обновляет состояние ракеты за один шаг симуляции
func (r *Rocket) UpdateState() {
	if r.FuelMass > 0 && r.ActiveEngines > 0 {
		// Рассчитываем суммарную тягу
		thrust := float64(r.ActiveEngines) * maxThrustPerEngine
		totalMass := r.Mass
		if r.SecondStage {
			// Если вторая ступень ещё прикреплена, учитываем её массу
			totalMass += secondStageMass
		}
		// Чистая сила (тяга - сила тяжести - сопротивление воздуха)
		netForce := thrust - (totalMass * gravity) + r.CalculateDragForce()
		// Ускорение = сила / масса
		r.Acceleration = netForce / totalMass

		// Обновляем скорость и высоту
		r.Velocity += r.Acceleration * timeStep
		r.Altitude += r.Velocity * timeStep

		// Простое горизонтальное движение для GPS-координат
		r.X += r.Velocity * 0.1 * timeStep
		r.Y += r.Velocity * 0.1 * timeStep

		// Расход топлива
		fuelConsumed := float64(r.ActiveEngines) * fuelBurnRate * timeStep
		r.FuelMass -= math.Min(fuelConsumed, r.FuelMass) // Обеспечиваем, чтобы топливо не ушло в минус
		r.Mass = emptyMass + r.FuelMass
	} else {
		// Если двигатели выключены или закончилось топливо, ракета свободно падает
		r.Acceleration = -gravity + r.CalculateDragForce()/r.Mass
		r.Velocity += r.Acceleration * timeStep
		r.Altitude += r.Velocity * timeStep
	}
}

func main() {
	// Создаем ракету с начальными параметрами
	rocket := Rocket{
		Position:      0,
		Velocity:      0,
		Acceleration:  0,
		Altitude:      0,
		Mass:          emptyMass + initialFuelMass,
		FuelMass:      initialFuelMass,
		ActiveEngines: totalEngines,
		SecondStage:   true,
		X:             -1000, // Начальная координата X (1 км от башни)
		Y:             -1000, // Начальная координата Y (1 км от башни)
	}

	phase := "launch" // Начальная фаза

	for rocket.Altitude >= 0 {
		// Обновляем состояние ракеты
		rocket.UpdateState()

		// Управление фазами полета
		switch phase {
		case "launch":
			// Если ракета достигает орбитальной высоты или скорости, переходим к отделению
			if rocket.Altitude > orbitalHeight || rocket.Velocity > orbitalVelocity {
				fmt.Println("Separating second stage...")
				rocket.SecondStage = false
				phase = "boostback"
			}
		case "boostback":
			// Снижаем количество активных двигателей для разворота
			rocket.ActiveEngines = totalEngines / 3
			if rocket.Velocity <= 0 {
				phase = "landing"
				fmt.Println("Reorienting for landing...")
			}
		case "landing":
			// Управление двигателями для мягкой посадки
			if rocket.Altitude > 100 {
				rocket.ActiveEngines = totalEngines / 10
			} else {
				rocket.ActiveEngines = int(math.Max(float64(totalEngines/15), 1))
			}
			// Навигация по GPS
			distanceToBeacon := math.Sqrt(math.Pow(beaconX-rocket.X, 2) + math.Pow(beaconY-rocket.Y, 2))
			if distanceToBeacon > 1 {
				// Простое корректирующее движение к башне
				rocket.X += -0.1 * (rocket.X - beaconX)
				rocket.Y += -0.1 * (rocket.Y - beaconY)
			}
		}

		// Вывод текущего состояния ракеты
		fmt.Printf("Phase: %s, Time: %.2fs, Altitude: %.2fm, Velocity: %.2fm/s, Fuel: %.2fkg, X: %.2fm, Y: %.2fm\n",
			phase, timeStep, rocket.Altitude, rocket.Velocity, rocket.FuelMass, rocket.X, rocket.Y)

		// Пауза для визуализации (опционально)
		time.Sleep(time.Duration(timeStep*1000) * time.Millisecond)

		// Прерывание при достижении земли
		if rocket.Altitude < 0 {
			fmt.Println("Landing complete.")
			// Проверка попадания на башню
			distanceToBeacon := math.Sqrt(math.Pow(beaconX-rocket.X, 2) + math.Pow(beaconY-rocket.Y, 2))
			if distanceToBeacon <= landingTolerance {
				fmt.Printf("Success! Landed on the tower. Distance to beacon: %.2fm\n", distanceToBeacon)
			} else {
				fmt.Printf("Missed the tower. Distance to beacon: %.2fm\n", distanceToBeacon)
			}
			break
		}
	}
}