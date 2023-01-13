export type Workout = {
    id: number;
    startDateEpochUtc: string;
};

export type Exercise = {
    id: number;
    name: string;
};

export type Set = {
    id: number;
    exerciseId: number;
    exerciseName: string;
    dateEpochUtc: string;
    repetitions: number;
    weight: number;
};

export type EditSet = {
    id: number | null;
    exerciseId: number;
    repetitions: number;
    weight: number;
};
