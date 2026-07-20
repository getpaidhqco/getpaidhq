import { Metadata } from './common';

/** Meter dimension filter (spec: CreateMeterRequest/MeterResponse `filters` items). */
export interface MeterFilter {
  field: string;
  values: string[];
}

/** Meter (spec: MeterResponse). */
export interface MeterResponse {
  aggregation: string;
  carry_over: boolean;
  code: string;
  created_at: string;
  field_name: string;
  filters: MeterFilter[];
  group_by: string[];
  id: string;
  metadata: Metadata;
  name: string;
  rounding_mode: string;
  rounding_scale: number;
  updated_at: string;
}

/** Create meter input (spec: CreateMeterRequest). */
export interface CreateMeterRequest {
  aggregation: string;
  carry_over?: boolean;
  code: string;
  field_name?: string;
  filters?: MeterFilter[];
  group_by?: string[];
  metadata?: Metadata;
  name: string;
  rounding_mode?: string;
  rounding_scale?: number;
}
