import { apiErrorMessage } from "../store";
import type { EditSet, Exercise, ExerciseExists, Set, Statistics, Workout } from "./types";

type SetEntity = {
    id: number;
    exerciseId: number;
    exerciseName: string;
    doneSecondsUnixEpoch: number;
    repetitions: number;
    weight: number;
};

class ApiService {
    private prefix = "/api";

    async getWorkoutList(): Promise<Workout[]> {
        type WorkoutEntity = {
            id: number;
            startSecondsUnixEpoch: number;
        };
        return (await this.getJson<WorkoutEntity[]>(`workouts`)).map(x => ({
            id: x.id,
            started: new Date(x.startSecondsUnixEpoch * 1000),
        }));
    }

    async deleteWorkout(id: number): Promise<void> {
        await this.getJson(
            `workouts/${id}`,
            {
                method: "DELETE",
            },
            false,
        );
    }

    async createWorkout(): Promise<number> {
        return (
            await this.getJson<{ id: number }>(`workouts`, {
                method: "POST",
            })
        ).id;
    }

    async getSetsByWorkoutId(id: number): Promise<Set[]> {
        return (await this.getJson<SetEntity[]>(`workouts/${id}/sets`)).map(x => ({
            id: x.id,
            exerciseId: x.exerciseId,
            exerciseName: x.exerciseName,
            repetitions: x.repetitions,
            weight: x.weight,
            date: new Date(x.doneSecondsUnixEpoch * 1000),
        }));
    }

    async getExercises(): Promise<Exercise[]> {
        return await this.getJson<Exercise[]>(`exercises`);
    }

    async getSetByIds(workoutId: number, setId: number): Promise<Set> {
        const set = await this.getJson<SetEntity>(`workouts/${workoutId}/sets/${setId}`);

        return {
            id: set.id,
            exerciseId: set.exerciseId,
            exerciseName: set.exerciseName,
            repetitions: set.repetitions,
            weight: set.weight,
            date: new Date(set.doneSecondsUnixEpoch * 1000),
        };
    }

    async createOrUpdateSet(workoutId: number, set: EditSet): Promise<void> {
        let promise: Promise<Response>;

        if (set.setId === null) {
            promise = this.getJson(
                `workouts/${workoutId}/sets`,
                {
                    method: "POST",
                    body: JSON.stringify(set),
                },
                false,
            );
        } else {
            promise = this.getJson(
                `workouts/${workoutId}/sets/${set.setId}`,
                {
                    method: "PUT",
                    body: JSON.stringify(set),
                },
                false,
            );
        }

        await promise;
    }

    async deleteSetById(workoutId: number, setId: number): Promise<void> {
        await this.getJson(
            `workouts/${workoutId}/sets/${setId}`,
            {
                method: "DELETE",
            },
            false,
        );
    }

    async getNewSetRecommendation(workoutId: number): Promise<Set> {
        return await this.getJson<Set>(`workouts/${workoutId}/sets/recommendation`);
    }

    async getStatistics(): Promise<Statistics> {
        return await this.getJson<Statistics>("statistics");
    }

    async existsExercise(name: string): Promise<ExerciseExists> {
        return await this.getJson<ExerciseExists>("exercises/exists", {
            method: "POST",
            body: JSON.stringify({ name }),
        });
    }

    async createExercise(name: string): Promise<Exercise> {
        return await this.getJson<Exercise>("exercises", {
            method: "POST",
            body: JSON.stringify({ name }),
        });
    }

    private async getJson<T>(
        url: RequestInfo,
        init: RequestInit = null,
        returnsJson: boolean = true,
    ): Promise<T | null> {
        try {
            if (init !== null) {
                init.headers = {
                    ...init.headers,
                    ["Content-Type"]: "application/json",
                };
            }

            const result = await fetch(`${this.prefix}/${url}`, init);

            if (!result.ok) {
                setApiErrorMessage("No connection to the server.");
                return null;
            }

            if (returnsJson) {
                return (await result.json()) as T;
            }
        } catch (err) {
            setApiErrorMessage(`Unexpected error: ${err}`);
            return null;
        }
    }
}

function setApiErrorMessage(message: string) {
    apiErrorMessage.set(message);
}

export const api = new ApiService();
