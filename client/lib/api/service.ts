import { apiErrorMessage } from "../store";
import type { EditSet, Exercise, Set, Workout } from "./types";

class ApiService {
    getWorkoutList(): Promise<Workout[]> {
        console.warn("Implement getWorkoutList");

        return new Promise((resolve, _) => {
            setTimeout(() => {
                resolve([
                    {
                        id: 1,
                        startDateEpochUtc: "2",
                    },
                    {
                        id: 3,
                        startDateEpochUtc: "4",
                    },
                    {
                        id: 5,
                        startDateEpochUtc: "6",
                    },
                ]);
            }, 1000);
        });
    }

    deleteWorkout(id: number): Promise<void> {
        console.warn("Implement deleteWorkout, id: ", id);

        return new Promise((resolve, _) => {
            setTimeout(resolve, 1000);
        });
    }

    createWorkout(): Promise<Workout> {
        console.warn("Implement createWorkout");

        return new Promise((resolve, _) => {
            setTimeout(
                () =>
                    resolve({
                        id: 15,
                        startDateEpochUtc: "12345678",
                    }),
                1000,
            );
        });
    }

    getSetsByWorkoutId(id: number): Promise<Set[]> {
        console.warn("Implement getSetsByWorkoutId, id: ", id);

        return new Promise((resolve, _) => {
            setTimeout(() => {
                resolve([
                    {
                        id: 1,
                        exerciseId: 1,
                        exerciseName: "Dehnen",
                        dateEpochUtc: "110923123",
                        repetitions: 1,
                        weight: 0,
                    },
                    {
                        id: 2,
                        exerciseId: 9,
                        exerciseName: "Handstand",
                        dateEpochUtc: "123234131",
                        repetitions: 3,
                        weight: 0,
                    },
                    {
                        id: 3,
                        exerciseId: 4,
                        exerciseName: "Squats",
                        dateEpochUtc: "123123131",
                        repetitions: 8,
                        weight: 100,
                    },
                    {
                        id: 4,
                        exerciseId: 7,
                        exerciseName: "Deadlifts",
                        dateEpochUtc: "4231231234",
                        repetitions: 8,
                        weight: 90,
                    },
                ]);
            }, 1000);
        });
    }

    getExercises(): Promise<Exercise[]> {
        console.warn("Implement getExercises");

        return new Promise((resolve, _) => {
            setTimeout(() => {
                resolve([
                    {
                        id: 1,
                        name: "Dehnen",
                    },
                    {
                        id: 2,
                        name: "Deadlift",
                    },
                    {
                        id: 3,
                        name: "Beinpresse",
                    },
                    {
                        id: 4,
                        name: "Handstand",
                    },
                    {
                        id: 5,
                        name: "Squats",
                    },
                ]);
            }, 1000);
        });
    }

    getSetById(id: number): Promise<Set> {
        console.warn("Implement getSetById, id: ", id);

        return new Promise((resolve, _) => {
            setTimeout(
                () =>
                    resolve({
                        id: 19,
                        dateEpochUtc: "1234567923",
                        exerciseId: 2,
                        exerciseName: "Deadlift",
                        repetitions: 12,
                        weight: 75,
                    }),
                1000,
            );
        });
    }

    saveSet(set: EditSet): Promise<void> {
        console.warn("Implement saveSet:", set);

        return new Promise((resolve, _) => {
            setTimeout(resolve, 1000);
        });
    }

    deleteSetById(id: number): Promise<void> {
        console.warn("Implement deleteSetById, id: ", id);

        return new Promise((resolve, _) => {
            setTimeout(resolve, 1000);
        });
    }
}

function setApiErrorMessage(message: string) {
    apiErrorMessage.set(message);
}

export const api = new ApiService();
