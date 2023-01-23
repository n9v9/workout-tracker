export type Workout = {
    id: number;
    started: Date;
};

export type Exercise = {
    id: number;
    name: string;
};

export type Set = {
    id: number;
    exerciseId: number;
    exerciseName: string;
    date: Date;
    repetitions: number;
    weight: number;
};

export type EditSet = {
    setId: number | null;
    exerciseId: number;
    repetitions: number;
    weight: number;
};

export type Statistics = {
    totalWorkouts: number;
    totalDurationSeconds: number;
    avgDurationSeconds: number;
    totalSets: number;
    totalReps: number;
    avgRepsPerSet: number;
};

export type ExerciseExists = {
    exists: boolean;
};

export type ExerciseCountInSets = {
    count: number;
};
