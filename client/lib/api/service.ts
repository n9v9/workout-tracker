import { apiErrorMessage } from "../store";
import type { EditSet, Exercise, Set, Workout } from "./types";

class ApiService {
    private prefix = "/api";

    async getWorkoutList(): Promise<Workout[]> {
        return await this.getJson<Workout[]>(`/workouts`);
    }

    async deleteWorkout(id: number): Promise<void> {
        await this.getJson(
            `/workouts/${id}`,
            {
                method: "DELETE",
            },
            false,
        );
    }

    async createWorkout(): Promise<number> {
        return (
            await this.getJson<{ id: number }>(`/workouts`, {
                method: "POST",
            })
        ).id;
    }

    async getSetsByWorkoutId(id: number): Promise<Set[]> {
        return await this.getJson<Set[]>(`/workouts/${id}/sets`);
    }

    async getExercises(): Promise<Exercise[]> {
        return await this.getJson<Exercise[]>(`/exercises`);
    }

    async getSetByIds(workoutId: number, setId: number): Promise<Set> {
        return await this.getJson<Set>(`/workouts/${workoutId}/sets/${setId}`);
    }

    async createOrUpdateSet(workoutId: number, set: EditSet): Promise<void> {
        let promise: Promise<Response>;

        if (set.setId === null) {
            promise = this.getJson(
                `/workouts/${workoutId}/sets`,
                {
                    method: "POST",
                    body: JSON.stringify(set),
                },
                false,
            );
        } else {
            promise = this.getJson(
                `/workouts/${workoutId}/sets/${set.setId}`,
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
            `/workouts/${workoutId}/sets/${setId}`,
            {
                method: "DELETE",
            },
            false,
        );
    }

    async getNewSetRecommendation(workoutId: number): Promise<Set> {
        return await this.getJson<Set>(`/workouts/${workoutId}/sets/recommendation`);
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
