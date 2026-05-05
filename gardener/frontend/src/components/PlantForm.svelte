<script>
  import { onDestroy } from 'svelte'

  export let plant = {}
  export let isNew = true
  export let onSave
  export let onCancel
  export let onFileSelected

  let formData = { ...plant }
  let selectedFile = null
  let previewUrl = null
  let photosToRemove = []

  function handleFileSelect(e) {
    const file = e.target.files[0]
    if (file) {
      selectedFile = file
      if (onFileSelected) onFileSelected(file)
      if (previewUrl) URL.revokeObjectURL(previewUrl)
      previewUrl = URL.createObjectURL(file)
    }
  }

  // ✅ Удалить фото из галереи
  function removePhoto(filename) {
    if (!photosToRemove.includes(filename)) {
      photosToRemove.push(filename)
    }
    try {
      const photos = JSON.parse(formData.photos || '[]')
      const updated = photos.filter((p) => p !== filename)
      formData.photos = JSON.stringify(updated)
    } catch {
      /* intentionally empty - ignore JSON parse errors */
    }
  }

  // ✅ Сделать фото обложкой (переместить на позицию 0)
  function setCoverPhoto(filename) {
    try {
      const photos = JSON.parse(formData.photos || '[]')
      const idx = photos.indexOf(filename)
      if (idx > 0) {
        // Удаляем с текущей позиции и вставляем в начало
        photos.splice(idx, 1)
        photos.unshift(filename)
        formData.photos = JSON.stringify(photos)
      }
    } catch {
      /* intentionally empty - ignore JSON parse errors */
    }
  }

  function submit() {
    let serial = (formData.serial_number || '').trim().toUpperCase()

    if (isNew) {
      if (!serial) {
        alert('№ наклейки обязателен')
        return
      }
      if (!/^[A-Za-z0-9\-_]{1,}$/.test(serial)) {
        alert('№ наклейки: только буквы, цифры, дефис или подчёркивание')
        return
      }
      formData.serial_number = serial
    }

    const payload = {
      ...formData,
      id: plant.id || 0,
      _original_serial: !isNew ? plant.serial_number : '',
      photos: formData.photos || '[]',
      photosToRemove: JSON.stringify(photosToRemove)
    }

    onSave(payload)
  }

  onDestroy(() => {
    if (previewUrl) URL.revokeObjectURL(previewUrl)
  })
</script>

<div class="overlay" on:click|self={onCancel}>
  <div class="modal">
    <h2>{isNew ? '➕ Новое растение' : '✏️ Редактирование'}</h2>

    <form on:submit|preventDefault={submit}>
      <input type="hidden" name="id" value={plant.id || 0} />
      <input type="hidden" name="_original_serial" value={!isNew ? plant.serial_number : ''} />

      <label
        >№ наклейки
        <input
          bind:value={formData.serial_number}
          required
          disabled={!isNew}
          placeholder={isNew ? 'напр: PKG-1, A1' : 'Не изменяется'}
          title={isNew
            ? 'Буквы, цифры, дефис или подчёркивание (мин. 1 символ)'
            : 'Поле недоступно для редактирования'}
        />
      </label>

      <label>Вид растения <input bind:value={formData.plant_type} /></label>
      <label>Откуда семена <input bind:value={formData.seed_source} /></label>
      <label>Описание <textarea bind:value={formData.description}></textarea></label>
      <label>Плоды <textarea bind:value={formData.fruit_description}></textarea></label>
      <label>Агротехника <textarea bind:value={formData.technique}></textarea></label>
      <label
        >Годы посадки <input bind:value={formData.planting_years} placeholder="2024, 2025" /></label
      >
      <label
        >Остаток семян (шт)
        <input
          type="number"
          min="0"
          step="1"
          bind:value={formData.seed_remainder}
          on:input={(e) => (formData.seed_remainder = Math.max(0, parseInt(e.target.value) || 0))}
        />
      </label>
      <label>Теги <input bind:value={formData.tags} placeholder="через запятую" /></label>

      <label>
        📷 Фото
        <input type="file" accept="image/*" on:change={handleFileSelect} />
        {#if previewUrl}
          <div class="preview-new">
            <small>Новое: {selectedFile?.name}</small>
            <img src={previewUrl} alt="Preview" class="preview-img" />
          </div>
        {/if}

        {#if formData.photos && JSON.parse(formData.photos || '[]').length > 0}
          <div class="existing-photos">
            <small>Текущие фото:</small>
            <div class="photo-grid">
              {#each JSON.parse(formData.photos || '[]') as photo (photo)}
                <!-- svelte-ignore a11y_click_events_have_key_events -->
                <!-- svelte-ignore a11y_no_static_element_interactions -->
                <div
                  class="photo-item"
                  class:is-cover={photo === (JSON.parse(formData.photos || '[]')[0] || '')}
                  role="presentation"
                >
                  <img
                    src={`/api/photos/${formData.serial_number}/${photo}`}
                    alt=""
                    class="thumb"
                    on:error={(e) => (e.target.style.display = 'none')}
                  />
                  <div class="photo-actions">
                    <button
                      type="button"
                      class="action-btn cover-btn"
                      on:click={() => setCoverPhoto(photo)}
                      title="Сделать обложкой">⭐</button
                    >
                    <button
                      type="button"
                      class="action-btn remove-btn"
                      on:click={() => removePhoto(photo)}
                      title="Удалить фото">×</button
                    >
                  </div>
                  {#if photo === (JSON.parse(formData.photos || '[]')[0] || '')}
                    <span class="cover-badge">Обложка</span>
                  {/if}
                </div>
              {/each}
            </div>
          </div>
        {/if}
      </label>

      <label>Комментарии <textarea bind:value={formData.comments}></textarea></label>

      <div class="actions">
        <button type="button" class="secondary" on:click={onCancel}>Отмена</button>
        <button type="submit">Сохранить</button>
      </div>
    </form>
  </div>
</div>

<style>
  .overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 100;
  }
  .modal {
    background: white;
    padding: 1.5rem;
    border-radius: 8px;
    width: 90%;
    max-width: 600px;
    max-height: 90vh;
    overflow-y: auto;
  }
  form {
    display: grid;
    gap: 0.8rem;
  }
  label {
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
    font-size: 0.9rem;
  }
  input,
  textarea {
    padding: 0.5rem;
    border: 1px solid #ccc;
    border-radius: 4px;
    font-size: 1rem;
  }
  textarea {
    resize: vertical;
    min-height: 60px;
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
    border-radius: 4px;
    cursor: pointer;
    font-size: 1rem;
  }
  button[type='submit'] {
    background: #4f46e5;
    color: white;
  }
  button.secondary {
    background: #e5e7eb;
    color: #111;
  }

  .preview-new {
    margin-top: 0.5rem;
  }
  .preview-img {
    max-width: 100%;
    max-height: 150px;
    border-radius: 4px;
  }

  .existing-photos {
    margin-top: 0.8rem;
  }
  .photo-grid {
    display: flex;
    gap: 0.5rem;
    flex-wrap: wrap;
    margin-top: 0.3rem;
  }

  .photo-item {
    position: relative;
    border: 2px solid transparent;
    border-radius: 6px;
    overflow: hidden;
  }
  .photo-item.is-cover {
    border-color: #f59e0b;
    box-shadow: 0 0 0 2px rgba(245, 158, 11, 0.3);
  }

  .photo-item .thumb {
    width: 80px;
    height: 80px;
    object-fit: cover;
    border-radius: 6px;
    display: block;
  }

  .photo-actions {
    position: absolute;
    top: 4px;
    right: 4px;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .action-btn {
    width: 20px;
    height: 20px;
    border-radius: 50%;
    border: none;
    font-size: 12px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 0;
    line-height: 1;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
  }

  .cover-btn {
    background: #fbbf24;
    color: #1f2937;
  }
  .cover-btn:hover {
    background: #f59e0b;
  }

  .remove-btn {
    background: #ef4444;
    color: white;
  }
  .remove-btn:hover {
    background: #dc2626;
  }

  .cover-badge {
    position: absolute;
    bottom: 4px;
    left: 4px;
    background: rgba(245, 158, 11, 0.95);
    color: #1f2937;
    font-size: 10px;
    font-weight: 600;
    padding: 2px 6px;
    border-radius: 10px;
    pointer-events: none;
  }
</style>
