import { apiErrorMessage, isLoading, uiDisabled } from "../store";
import type {
    EditSet,
    Exercise,
    ExerciseCountInSets,
    ExerciseExists,
    ExerciseSet,
    Statistics,
    Workout,
} from "./types";

type SetEntity = {
    id: number;
    exerciseId: number;
    exerciseName: string;
    doneSecondsUnixEpoch: number;
    repetitions: number;
    weight: number;
    note: string | null;
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

    async getSetsByWorkoutId(id: number): Promise<ExerciseSet[]> {
        return (await this.getJson<SetEntity[]>(`workouts/${id}/sets`)).map(x => ({
            id: x.id,
            exerciseId: x.exerciseId,
            exerciseName: x.exerciseName,
            repetitions: x.repetitions,
            weight: x.weight,
            date: new Date(x.doneSecondsUnixEpoch * 1000),
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
            repetitions: set.repetitions,
            weight: set.weight,
            date: new Date(set.doneSecondsUnixEpoch * 1000),
            note: set.note,
        };
    }

    async createOrUpdateSet(workoutId: number, setId: null | number, set: EditSet): Promise<void> {
        let promise: Promise<Response>;

        if (setId === null) {
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
                `sets/${setId}`,
                {
                    method: "PUT",
                    body: JSON.stringify(set),
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

    async getNewSetRecommendation(workoutId: number): Promise<ExerciseSet> {
        return await this.getJson<ExerciseSet>(`workouts/${workoutId}/sets/recommendation`);
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
        init: RequestInit = null,
        returnsJson: boolean = true,
    ): Promise<T | null> {
        uiDisabled.set(true);
        isLoading.set(true);

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
        } finally {
            uiDisabled.set(false);
            isLoading.set(false);
        }
    }
}

function setApiErrorMessage(message: string) {
    apiErrorMessage.set(message);
}

export const api = new ApiService();
