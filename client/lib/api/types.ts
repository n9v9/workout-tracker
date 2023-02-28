export type Workout = {
    id: number;
    started: Date;
    note: string;
};

export type Exercise = {
    id: number;
    name: string;
};

export type ExerciseSet = {
    id: number;
    exerciseId: number;
    exerciseName: string;
    workoutId: number;
    date: Date;
    repetitions: number;
    weight: number;
    note: string | null | undefined;
};

export type EditSet = {
    exerciseId: number;
    repetitions: number;
    weight: number;
    note: string | null;
};

export type Statistics = {
    totalWorkouts: number;
    totalDurationSeconds: number;
    avgDurationSeconds: number;
    totalSets: number;
    totalReps: number;
    avgRepsPerSet: number;
};

export type ExerciseCountInSets = {
    count: number;
};
