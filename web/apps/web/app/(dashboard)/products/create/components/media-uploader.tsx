"use client"

import type React from "react"

import { useState } from "react"
import Image from "next/image"
import { Upload, X } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import { H3 } from "@/components/ui/typography"

interface MediaUploaderProps {
  media: string[]
  setMedia: React.Dispatch<React.SetStateAction<string[]>>
}

export default function MediaUploader({ media, setMedia }: MediaUploaderProps) {
  const [dragActive, setDragActive] = useState(false)

  const handleDrag = (e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    if (e.type === "dragenter" || e.type === "dragover") {
      setDragActive(true)
    } else if (e.type === "dragleave") {
      setDragActive(false)
    }
  }

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setDragActive(false)

    if (e.dataTransfer.files && e.dataTransfer.files[0]) {
      handleFiles(e.dataTransfer.files)
    }
  }

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    e.preventDefault()
    if (e.target.files && e.target.files[0]) {
      handleFiles(e.target.files)
    }
  }

  const handleFiles = (files: FileList) => {
    // In a real app, you would upload these files to your server or cloud storage
    // For this example, we'll just create object URLs
    const newMedia = Array.from(files).map((file) => URL.createObjectURL(file))
    setMedia([...media, ...newMedia])
  }

  const removeMedia = (index: number) => {
    const newMedia = [...media]
    newMedia.splice(index, 1)
    setMedia(newMedia)
  }

  return (
    <div className="space-y-4">
      <div
        className={`border-2 border-dashed rounded-lg p-8 text-center ${
          dragActive ? "border-primary bg-primary/5" : "border-muted-foreground/25"
        }`}
        onDragEnter={handleDrag}
        onDragLeave={handleDrag}
        onDragOver={handleDrag}
        onDrop={handleDrop}
      >
        <div className="flex flex-col items-center justify-center gap-2">
          <Upload className="h-10 w-10 text-muted-foreground" />
          <H3>Drag and drop your images here</H3>
          <p className="text-sm text-muted-foreground">or click to browse (PNG, JPG, GIF up to 5MB)</p>
          <Button variant="outline" className="mt-2" onClick={() => document.getElementById("file-upload")?.click()}>
            Select Files
          </Button>
          <input id="file-upload" type="file" multiple accept="image/*" className="hidden" onChange={handleChange} />
        </div>
      </div>

      {media.length > 0 && (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mt-6">
          {media.map((src, index) => (
            <Card key={index} className="relative group overflow-hidden">
              <div className="aspect-square relative">
                <Image
                  src={src || "/placeholder.svg"}
                  alt={`Product image ${index + 1}`}
                  fill
                  className="object-cover"
                />
                <button
                  type="button"
                  onClick={() => removeMedia(index)}
                  className="absolute top-2 right-2 bg-background/80 p-1 rounded-full opacity-0 group-hover:opacity-100 transition-opacity"
                >
                  <X className="h-4 w-4" />
                </button>
              </div>
            </Card>
          ))}
        </div>
      )}
    </div>
  )
}
