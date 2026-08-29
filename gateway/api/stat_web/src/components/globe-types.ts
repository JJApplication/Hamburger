export type GlobeGeometry =
  | { type: "Polygon"; coordinates: number[][][] }
  | { type: "MultiPolygon"; coordinates: number[][][][] };

export interface GlobeCountry {
  id: string;
  code: string;
  requests: number;
  center: [number, number];
  geometry: GlobeGeometry;
  properties: {
    name?: string;
  };
}

export interface GlobePoint {
  id: string;
  code: string;
  requests: number;
  lat: number;
  lng: number;
}
