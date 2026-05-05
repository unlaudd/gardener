export async function fetchPlants(query = '', sort = 'plant_type') {
  const params = new URLSearchParams({ q: query, sort })
  const res = await fetch(`/api/plants?${params}`)
  if (!res.ok) throw new Error('API error')
  return res.json()
}

export async function createPlant(data) {
  const res = await fetch('/api/plants', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data)
  })
  if (!res.ok) throw new Error('Create failed')
  return res.json()
}
