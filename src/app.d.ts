// src/app.d.ts
declare global {
  namespace App {
    interface Locals {
      user: {
        id: number;
        email: string;
        name: string;
        is_admin: boolean;
      } | null;
    }
  }
}
export {};
