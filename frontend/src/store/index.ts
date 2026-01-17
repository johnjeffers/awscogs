import { configureStore } from '@reduxjs/toolkit';
import costReducer from './costSlice';

export const store = configureStore({
  reducer: {
    costs: costReducer,
  },
});

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;
