/** Setting (spec: SettingResponse). */
export interface SettingResponse {
  created_at: string;
  id: string;
  parent_id: string;
  type: string;
  updated_at: string;
  value: string;
}

/** Create setting input (spec: CreateSettingRequest). */
export interface CreateSettingRequest {
  id: string;
  parent_id?: string;
  type?: string;
  value?: string;
}

/** Update setting input (spec: UpdateSettingRequest). */
export interface UpdateSettingRequest {
  type?: string;
  value?: string;
}
