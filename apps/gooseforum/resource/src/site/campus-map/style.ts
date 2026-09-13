import type {
  ExpressionSpecification,
  LayerSpecification,
  StyleSpecification,
} from 'maplibre-gl'
import type { CampusData } from './catalog'
import { makeSportsDetails } from './sports-details'

// Cartographic colors are local to this map, not changes to the shared UI tokens.
export const categoryColors = {
  academic: '#c8ac85',
  sport: '#cc7f60',
  library: '#b39070',
  food: '#c89955',
  living: '#9ca7c0',
  place: '#739b82',
}
const buildingColor: ExpressionSpecification = [
  'match',
  ['get', 'category'],
  'library',
  '#dfb58e',
  'food',
  '#e8c990',
  'living',
  '#c5cddb',
  'sport',
  '#e4b89e',
  'place',
  '#d6d8cf',
  '#ede3cc',
]
const sportsColor: ExpressionSpecification = [
  'match',
  ['get', 'sport'],
  'basketball',
  '#d8a185',
  'tennis',
  '#82b3ac',
  'running',
  '#cb927e',
  'badminton',
  '#90b5ab',
  '#9abb87',
]
const polygons: ExpressionSpecification = ['==', ['geometry-type'], 'Polygon']
export const selectableLayers = [
  'buildings',
  'building-roofs',
  'sports',
  'poi-hit',
]

export function makeMapStyle(
  data: CampusData,
  dimensional: boolean,
): StyleSpecification {
  const layers: LayerSpecification[] = [
    {
      id: 'background',
      type: 'background',
      paint: { 'background-color': '#f0eee5' },
    },
    {
      id: 'grounds',
      type: 'fill',
      source: 'campus',
      filter: [
        'all',
        polygons,
        ['==', ['get', 'amenity'], 'university'],
        ['!', ['has', 'building']],
      ],
      paint: { 'fill-color': '#e4e9d4' },
    },
    {
      id: 'greens',
      type: 'fill',
      source: 'campus',
      filter: ['all', polygons, ['==', ['get', 'category'], 'green']],
      paint: { 'fill-color': '#c1d7ac', 'fill-opacity': 0.9 },
    },
    {
      id: 'water',
      type: 'fill',
      source: 'campus',
      filter: ['all', polygons, ['==', ['get', 'category'], 'water']],
      paint: { 'fill-color': '#a3d2d8', 'fill-outline-color': '#84bdc6' },
    },
    {
      id: 'waterways',
      type: 'line',
      source: 'campus',
      filter: [
        'all',
        ['==', ['geometry-type'], 'LineString'],
        ['has', 'waterway'],
      ],
      paint: {
        'line-color': '#a3d2d8',
        'line-width': ['interpolate', ['linear'], ['zoom'], 15, 2, 19, 12],
      },
    },
    {
      id: 'road-edge',
      type: 'line',
      source: 'campus',
      filter: ['has', 'highway'],
      layout: { 'line-cap': 'round', 'line-join': 'round' },
      paint: {
        'line-color': '#d5d6c7',
        'line-width': ['interpolate', ['linear'], ['zoom'], 15, 3, 19, 21],
      },
    },
    {
      id: 'roads',
      type: 'line',
      source: 'campus',
      filter: ['has', 'highway'],
      layout: { 'line-cap': 'round', 'line-join': 'round' },
      paint: {
        'line-color': '#fffdf4',
        'line-width': ['interpolate', ['linear'], ['zoom'], 15, 2, 19, 17],
      },
    },
    {
      id: 'sports',
      type: 'fill',
      source: 'campus',
      filter: [
        'all',
        polygons,
        ['==', ['get', 'category'], 'sport'],
        ['!', ['has', 'building']],
      ],
      paint: { 'fill-color': sportsColor, 'fill-outline-color': '#ffffff' },
    },
    {
      id: 'sport-boundaries',
      type: 'line',
      source: 'campus',
      filter: [
        'all',
        polygons,
        ['==', ['get', 'category'], 'sport'],
        ['!', ['has', 'building']],
      ],
      paint: {
        'line-color': '#fffdf4',
        'line-width': 1.6,
        'line-opacity': 0.85,
      },
    },
    ...(['track', 'infield', 'stripe'] as const).map((part) => ({
      id: `sports-${part}`,
      type: 'fill' as const,
      source: 'sports-detail',
      filter: ['==', ['get', 'part'], part] as ExpressionSpecification,
      paint: {
        'fill-color': {
          track: '#bb675b',
          infield: '#84b078',
          stripe: '#91b980',
        }[part],
      },
    })),
    {
      id: 'sports-markings',
      type: 'line',
      source: 'sports-detail',
      filter: ['in', ['get', 'part'], ['literal', ['lane', 'marking']]],
      paint: {
        'line-color': '#fffbed',
        'line-opacity': 0.8,
        'line-width': [
          'interpolate',
          ['linear'],
          ['zoom'],
          15,
          0.35,
          18,
          1,
          20,
          2,
        ],
      },
    },
    {
      id: 'building-shadows',
      type: 'fill',
      source: 'campus',
      filter: ['has', 'building'],
      paint: {
        'fill-color': '#707b60',
        'fill-opacity': 0.13,
        'fill-translate': [3, 5],
      },
    },
    {
      id: 'building-roofs',
      type: 'fill',
      source: 'campus',
      filter: ['has', 'building'],
      paint: { 'fill-color': buildingColor, 'fill-outline-color': '#bfb69f' },
      layout: { visibility: dimensional ? 'none' : 'visible' },
    },
    {
      id: 'buildings',
      type: 'fill-extrusion',
      source: 'campus',
      filter: ['has', 'building'],
      paint: {
        'fill-extrusion-color': buildingColor,
        'fill-extrusion-height': [
          'min',
          34,
          ['max', 6, ['*', 2.7, ['to-number', ['get', 'building:levels'], 2]]],
        ],
        'fill-extrusion-opacity': 1,
      },
      layout: { visibility: dimensional ? 'visible' : 'none' },
    },
    {
      id: 'trees-shadow',
      type: 'circle',
      source: 'campus',
      filter: ['==', ['get', 'natural'], 'tree'],
      paint: {
        'circle-color': '#82976b',
        'circle-opacity': 0.2,
        'circle-translate': [2, 3],
        'circle-radius': ['interpolate', ['linear'], ['zoom'], 15, 2, 19, 9],
      },
    },
    {
      id: 'trees',
      type: 'circle',
      source: 'campus',
      filter: ['==', ['get', 'natural'], 'tree'],
      paint: {
        'circle-color': '#9abb80',
        'circle-stroke-color': '#84a56e',
        'circle-stroke-width': 0.8,
        'circle-radius': ['interpolate', ['linear'], ['zoom'], 15, 2, 19, 8],
      },
    },
    {
      id: 'poi-hit',
      type: 'circle',
      source: 'campus',
      filter: [
        'all',
        ['==', ['geometry-type'], 'Point'],
        ['has', 'name'],
        ['has', 'amenity'],
      ],
      paint: { 'circle-radius': 8, 'circle-opacity': 0 },
    },
    {
      id: 'selected-area',
      type: 'fill',
      source: 'selection',
      filter: polygons,
      paint: { 'fill-color': '#28654c', 'fill-opacity': 0.17 },
    },
    {
      id: 'selected-outline',
      type: 'line',
      source: 'selection',
      filter: polygons,
      paint: {
        'line-color': '#28654c',
        'line-width': 2.5,
        'line-dasharray': [3, 2],
      },
    },
  ]
  return {
    version: 8,
    light: {
      anchor: 'viewport',
      color: '#fff9e9',
      intensity: 0.28,
      position: [1.5, 200, 35],
    },
    sources: {
      campus: {
        type: 'geojson',
        data,
        attribution: '© OpenStreetMap contributors',
      },
      location: {
        type: 'geojson',
        data: { type: 'FeatureCollection', features: [] },
      },
      'sports-detail': { type: 'geojson', data: makeSportsDetails(data) },
      selection: {
        type: 'geojson',
        data: { type: 'FeatureCollection', features: [] },
      },
    },
    layers: [
      ...layers,
      {
        id: 'location-accuracy',
        type: 'fill',
        source: 'location',
        filter: ['==', ['get', 'part'], 'accuracy'],
        paint: { 'fill-color': '#428ce5', 'fill-opacity': 0.14 },
      },
      {
        id: 'location-accuracy-edge',
        type: 'line',
        source: 'location',
        filter: ['==', ['get', 'part'], 'accuracy'],
        paint: {
          'line-color': '#428ce5',
          'line-opacity': 0.45,
          'line-width': 1,
        },
      },
      {
        id: 'location-dot',
        type: 'circle',
        source: 'location',
        filter: ['==', ['get', 'part'], 'position'],
        paint: {
          'circle-color': '#2779e3',
          'circle-radius': 7,
          'circle-stroke-color': '#ffffff',
          'circle-stroke-width': 3,
        },
      },
    ],
  }
}
