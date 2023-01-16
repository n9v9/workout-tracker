import SetForm from "../components/SetForm.svelte";
import { apiErrorMessage } from "../store";
import type { EditSet, Exercise, Set, Workout } from "./types";

class ApiService {
    private prefix = "/api";

    async getWorkoutList(): Promise<Workout[]> {
        try {
            const result = await fetch(`${this.prefix}/workouts`);
            const json = await result.json();
            return json as Workout[];
        } catch (err) {
            setApiErrorMessage(err);
        }
    }

    async deleteWorkout(id: number): Promise<void> {
        try {
            await fetch(`${this.prefix}/workouts/${id}`, {
                method: "DELETE",
            });
        } catch (err) {
            setApiErrorMessage(err);
        }
    }

    async createWorkout(): Promise<number> {
        try {
            const result = await fetch(`${this.prefix}/workouts`, {
                method: "POST",
            });
            const json = await result.json();
            return json.id;
        } catch (err) {
            setApiErrorMessage(err);
        }
    }

    async getSetsByWorkoutId(id: number): Promise<Set[]> {
        try {
            const result = await fetch(`${this.prefix}/workouts/${id}/sets`);
            const json = await result.json();
            return json as Set[];
        } catch (err) {
            setApiErrorMessage(err);
        }
    }

    async getExercises(): Promise<Exercise[]> {
        try {
            const result = await fetch(`${this.prefix}/exercises`);
            const json = await result.json();
            return json as Exercise[];
        } catch (err) {
            setApiErrorMessage(err);
        }
    }

    async getSetByIds(workoutId: number, setId: number): Promise<Set> {
        try {
            const result = await fetch(`${this.prefix}/workouts/${workoutId}/sets/${setId}`);
            const json = await result.json();
            return json as Set;
        } catch (err) {
            setApiErrorMessage(err);
        }
    }

    async createOrUpdateSet(workoutId: number, set: EditSet): Promise<void> {
        let promise: Promise<Response>;

        if (set.setId === null) {
            promise = fetch(`${this.prefix}/workouts/${workoutId}/sets`, {
                method: "POST",
                body: JSON.stringify(set),
            });
        } else {
            promise = fetch(`${this.prefix}/workouts/${workoutId}/sets/${set.setId}`, {
                method: "PUT",
                body: JSON.stringify(set),
            });
        }

        try {
            await promise;
        } catch (err) {
            setApiErrorMessage(err);
        }
    }

    async deleteSetById(workoutId: number, setId: number): Promise<void> {
        try {
            await fetch(`${this.prefix}/workouts/${workoutId}/sets/${setId}`, {
                method: "DELETE",
            });
        } catch (err) {
            setApiErrorMessage(err);
        }
    }

    async getNewSetRecommendation(workoutId: number): Promise<Set> {
        try {
            const result = await fetch(`${this.prefix}/workouts/${workoutId}/sets/recommendation`);
            const json = await result.json();
            return json as Set;
        } catch (err) {
            setApiErrorMessage(err);
        }
    }
}

function setApiErrorMessage(message: string) {
    apiErrorMessage.set(message);
}

export const api = new ApiService();
