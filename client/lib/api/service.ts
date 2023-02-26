import { apiErrorMessage, isLoading, uiDisabled } from "../store";
import type {
    EditSet,
    Exercise,
    ExerciseCountInSets,
    ExerciseSet,
    Statistics,
    Workout,
} from "./types";

type SetEntity = {
    id: number;
    workoutId: number;
    exerciseId: number;
    exerciseName: string;
    createdUtcSeconds: number;
    repetitions: number;
    weight: number;
    note: string | null;
};

class ApiService {
    private prefix = "/api";

    async getWorkoutList(): Promise<Workout[]> {
        type WorkoutEntity = {
            id: number;
            createdUtcSeconds: number;
        };

        const workouts = (await this.getJson<WorkoutEntity[]>(`workouts`)).map(x => ({
            id: x.id,
            started: new Date(x.createdUtcSeconds * 1000),
        }));

        workouts.sort((a, b) => b.started.getTime() - a.started.getTime());

        return workouts;
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

    async getSetsByWorkoutId(id: number): Promise<ExerciseSet[]> {
        return (await this.getJson<SetEntity[]>(`workouts/${id}/sets`)).map(x => ({
            id: x.id,
            exerciseId: x.exerciseId,
            exerciseName: x.exerciseName,
            workoutId: x.workoutId,
            repetitions: x.repetitions,
            weight: x.weight,
            date: new Date(x.createdUtcSeconds * 1000),
            note: x.note ?? "",
        }));
    }

    async getSetsByExerciseId(id: number): Promise<ExerciseSet[]> {
        return (await this.getJson<SetEntity[]>(`exercises/${id}/sets`)).map(x => ({
            id: x.id,
            exerciseId: x.exerciseId,
            exerciseName: x.exerciseName,
            workoutId: x.workoutId,
            repetitions: x.repetitions,
            weight: x.weight,
            date: new Date(x.createdUtcSeconds * 1000),
            note: x.note ?? "",
        }));
    }

    async getExercises(): Promise<Exercise[]> {
        return await this.getJson<Exercise[]>(`exercises`);
    }

    async getSetByIds(setId: number): Promise<ExerciseSet> {
        const set = await this.getJson<SetEntity>(`sets/${setId}`);

        return {
            id: set.id,
            exerciseId: set.exerciseId,
            exerciseName: set.exerciseName,
            workoutId: set.workoutId,
            repetitions: set.repetitions,
            weight: set.weight,
            date: new Date(set.createdUtcSeconds * 1000),
            note: set.note,
        };
    }

    async createOrUpdateSet(workoutId: number, setId: null | number, set: EditSet): Promise<void> {
        let promise: Promise<Response>;

        if (setId === null) {
            promise = this.getJson(
                `sets`,
                {
                    method: "POST",
                    body: JSON.stringify({
                        workoutId,
                        ...set,
                    }),
                },
                false,
            );
        } else {
            promise = this.getJson(
                `sets/${setId}`,
                {
                    method: "PUT",
                    body: JSON.stringify({
                        workoutId,
                        ...set,
                    }),
                },
                false,
            );
        }

        await promise;
    }

    async deleteSetById(setId: number): Promise<void> {
        await this.getJson(
            `sets/${setId}`,
            {
                method: "DELETE",
            },
            false,
        );
    }

    async suggestNewSet(workoutId: number, exerciseId: number | null = null): Promise<ExerciseSet> {
        return await this.getJson<ExerciseSet>(`workouts/${workoutId}/sets/suggest`, {
            method: "POST",
            body: JSON.stringify({ exerciseId }),
        });
    }

    async getStatistics(): Promise<Statistics> {
        return await this.getJson<Statistics>("statistics");
    }

    async existsExercise(name: string): Promise<boolean> {
        name = name.toLowerCase().trim();

        const exercises = await this.getExercises();

        for (const exercise of exercises) {
            if (exercise.name.toLowerCase().trim() == name) {
                return true;
            }
        }

        return false;
    }

    async createExercise(name: string): Promise<Exercise> {
        return await this.getJson<Exercise>("exercises", {
            method: "POST",
            body: JSON.stringify({ name }),
        });
    }

    async updateExercise(id: number, name: string): Promise<Exercise> {
        return await this.getJson<Exercise>(`exercises/${id}`, {
            method: "PUT",
            body: JSON.stringify({ name }),
        });
    }

    async deleteExercise(id: number): Promise<void> {
        await this.getJson<void>(
            `exercises/${id}`,
            {
                method: "DELETE",
            },
            false,
        );
    }

    async getExerciseCountInSets(id: number): Promise<ExerciseCountInSets> {
        return await this.getJson<ExerciseCountInSets>(`exercises/${id}/count`);
    }

    private async getJson<T>(
        url: RequestInfo,
        init: RequestInit | null = null,
        returnsJson: boolean = true,
    ): Promise<T> {
        uiDisabled.set(true);
        isLoading.set(true);

        if (init === null) {
            init = {};
        }

        try {
            init.headers = {
                ...init.headers,
                ["Content-Type"]: "application/json",
            };

            const result = await fetch(`${this.prefix}/${url}`, init);

            if (!result.ok) {
                setApiErrorMessage("No connection to the server.");
                return null as T;
            }

            if (returnsJson) {
                return (await result.json()) as T;
            }
        } catch (err) {
            setApiErrorMessage(`Unexpected error: ${err}`);
            return null as T;
        } finally {
            uiDisabled.set(false);
            isLoading.set(false);
        }

        return null as T;
    }
}

function setApiErrorMessage(message: string) {
    apiErrorMessage.set(message);
}

export const api = new ApiService();
