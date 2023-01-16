export type Workout = {
    id: number;
    startedUtc: string;
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
    setId: number | null;
    exerciseId: number;
    repetitions: number;
    weight: number;
};
