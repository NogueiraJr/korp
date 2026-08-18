export interface LoginResponse {
  token: string;
  username: string;
}

export interface HealthStatus {
  status: string;
  service: string;
}

export interface ControlStatus {
  estoque: { running: boolean };
  faturamento: { running: boolean };
}

export type ServiceSource = 'estoque' | 'faturamento';

export interface ActivityLog {
  id: number;
  event: string;
  entity: string;
  details?: Record<string, unknown> | null;
  created_at: string;
}