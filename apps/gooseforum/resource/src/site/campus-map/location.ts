import type { FeatureCollection } from 'geojson'
export interface MapLocation {
  longitude: number
  latitude: number
  accuracy: number
  timestamp: number
}
export type LocationFailure =
  | 'unsupported'
  | 'insecure'
  | 'denied'
  | 'unavailable'
  | 'timeout'
export function requestLocation(
  geolocation: Geolocation | undefined,
  secure: boolean,
): Promise<MapLocation> {
  return new Promise((resolve, reject) => {
    if (!secure) return reject('insecure' satisfies LocationFailure)
    if (!geolocation) return reject('unsupported' satisfies LocationFailure)
    geolocation.getCurrentPosition(
      (position) => {
        const { longitude, latitude, accuracy } = position.coords
        if (
          !Number.isFinite(longitude) ||
          !Number.isFinite(latitude) ||
          !Number.isFinite(accuracy) ||
          Math.abs(latitude) > 90 ||
          Math.abs(longitude) > 180 ||
          accuracy < 0
        )
          return reject('unavailable' satisfies LocationFailure)
        resolve({
          longitude,
          latitude,
          accuracy,
          timestamp: position.timestamp,
        })
      },
      (error) =>
        reject(
          (
            { 1: 'denied', 2: 'unavailable', 3: 'timeout' } as Record<
              number,
              LocationFailure
            >
          )[error.code] ?? 'unavailable',
        ),
      { enableHighAccuracy: true, timeout: 15000, maximumAge: 30000 },
    )
  })
}
// Geodesic accuracy ring in WGS84, including high latitude and date-line fixes.
export function locationData(location: MapLocation | null): FeatureCollection {
  if (!location) return { type: 'FeatureCollection', features: [] }
  const { longitude, latitude, accuracy } = location,
    phi = (latitude * Math.PI) / 180,
    lambda = (longitude * Math.PI) / 180,
    d = accuracy / 6371008.8
  const ring = Array.from({ length: 65 }, (_, i) => {
    const b = (i * Math.PI) / 32,
      lat = Math.asin(
        Math.sin(phi) * Math.cos(d) + Math.cos(phi) * Math.sin(d) * Math.cos(b),
      )
    const lon =
      lambda +
      Math.atan2(
        Math.sin(b) * Math.sin(d) * Math.cos(phi),
        Math.cos(d) - Math.sin(phi) * Math.sin(lat),
      )
    return [(lon * 180) / Math.PI, (lat * 180) / Math.PI]
  })
  return {
    type: 'FeatureCollection',
    features: [
      {
        type: 'Feature',
        properties: { part: 'accuracy' },
        geometry: { type: 'Polygon', coordinates: [ring] },
      },
      {
        type: 'Feature',
        properties: { part: 'position' },
        geometry: { type: 'Point', coordinates: [longitude, latitude] },
      },
    ],
  }
}
