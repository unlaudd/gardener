<script>
  export let plants = []
  export let isOpen = false
  export let onClose

  const allFields = [
    { key: 'serial_number', label: '№ наклейки' },
    { key: 'plant_type', label: 'Вид растения' },
    { key: 'seed_source', label: 'Откуда семена' },
    { key: 'description', label: 'Описание' },
    { key: 'fruit_description', label: 'Описание плодов' },
    { key: 'technique', label: 'Агротехника' },
    { key: 'planting_years', label: 'Годы посадки' },
    { key: 'seed_remainder', label: 'Остаток семян' },
    { key: 'tags', label: 'Теги' },
    { key: 'comments', label: 'Комментарии' }
  ]

  let selectedFields = allFields.map((f) => f.key)
  let format = 'csv'

  // ✅ Вычисляем состояние чекбокса "Выбрать всё" (только для чтения)
  $: allSelected = selectedFields.length === allFields.length

  function toggleAll(e) {
    const checked = e.target.checked
    selectedFields = checked ? allFields.map((f) => f.key) : []
  }

  function doExport() {
    if (selectedFields.length === 0) {
      alert('Выберите хотя бы одно поле для выгрузки')
      return
    }
    if (plants.length === 0) {
      alert('Нет данных для экспорта')
      return
    }

    const fields = allFields.filter((f) => selectedFields.includes(f.key))
    let content = ''

    if (format === 'csv') {
      content += fields.map((f) => `"${f.label}"`).join(',') + '\n'
      content += plants
        .map((p) =>
          fields
            .map((f) => {
              let val = p[f.key] ?? ''
              val = val.toString().replace(/"/g, '""')
              return `"${val}"`
            })
            .join(',')
        )
        .join('\n')
    } else {
      const pad = 25
      content += fields.map((f) => f.label.padEnd(pad)).join(' | ') + '\n'
      content += fields.map(() => '-'.repeat(pad)).join('-+-') + '\n'
      content += plants
        .map((p) => fields.map((f) => (p[f.key] ?? '—').toString().padEnd(pad)).join(' | '))
        .join('\n')
    }

    const bom = '\uFEFF'
    const blob = new Blob([bom + content], {
      type: format === 'csv' ? 'text/csv;charset=utf-8' : 'text/plain;charset=utf-8'
    })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `gardener_export_${new Date().toISOString().slice(0, 10)}.${format}`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
    onClose()
  }
</script>

{#if isOpen}
  <!-- ✅ Добавлены role, tabindex и on:keydown для доступности -->
  <div
    class="overlay"
    on:click|self={onClose}
    on:keydown={(e) => {
      if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault()
        onClose()
      }
    }}
    role="button"
    tabindex="0"
  >
    <div class="dialog">
      <h2>📤 Экспорт данных</h2>

      <fieldset>
        <legend>Поля для выгрузки:</legend>
        <label class="all-toggle">
          <input type="checkbox" checked={allSelected} on:change={toggleAll} />
          Выбрать всё
        </label>
        <div class="fields-grid">
          {#each allFields as field}
            <label>
              <input type="checkbox" bind:group={selectedFields} value={field.key} />
              {field.label}
            </label>
          {/each}
        </div>
      </fieldset>

      <fieldset>
        <legend>Формат файла:</legend>
        <label><input type="radio" bind:group={format} value="csv" /> CSV (таблица)</label>
        <label><input type="radio" bind:group={format} value="txt" /> TXT (текст)</label>
      </fieldset>

      <div class="actions">
        <button class="secondary" on:click={onClose}>Отмена</button>
        <button class="primary" on:click={doExport}>Скачать файл</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.4);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 100;
  }
  .dialog {
    background: white;
    padding: 1.5rem;
    border-radius: 12px;
    width: 90%;
    max-width: 500px;
    max-height: 90vh;
    overflow-y: auto;
    box-shadow: 0 8px 30px rgba(0, 0, 0, 0.15);
  }
  h2 {
    margin: 0 0 1rem;
    font-size: 1.2rem;
  }
  fieldset {
    border: 1px solid #e2e8f0;
    border-radius: 8px;
    padding: 1rem;
    margin-bottom: 1rem;
  }
  legend {
    font-weight: 600;
    padding: 0 0.5rem;
    color: #475569;
  }
  .all-toggle {
    margin-bottom: 0.5rem;
    display: block;
    font-weight: 500;
  }
  .fields-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
    gap: 0.4rem;
  }
  label {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    cursor: pointer;
    font-size: 0.9rem;
  }
  .actions {
    display: flex;
    gap: 0.5rem;
    justify-content: flex-end;
    margin-top: 1rem;
  }
  button {
    padding: 0.6rem 1.2rem;
    border: none;
    border-radius: 6px;
    cursor: pointer;
    font-size: 0.95rem;
  }
  button.primary {
    background: #4f46e5;
    color: white;
  }
  button.secondary {
    background: #e5e7eb;
    color: #111;
  }
</style>
