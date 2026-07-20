'use client'

import {Input} from '@/components/ui/input'
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select'
import {countries} from '@/lib/countries'

import {useState} from 'react'

const uniqueCountries = Array.from(
  new Map(countries.map((c) => [c.code, c])).values(),
).sort((a, b) => a.name.localeCompare(b.name))

export function Address() {
  const [country, setCountry] = useState(uniqueCountries[0])

  return (
    <div className="grid grid-cols-2 gap-6">
      <Input
        aria-label="Street Address"
        name="address"
        placeholder="Street Address"
        defaultValue="147 Catalyst Ave"
        className="col-span-2"
      />
      <Input aria-label="City" name="city" placeholder="City" defaultValue="Toronto" className="col-span-2"/>

      <Input aria-label="Postal code" name="postal_code" placeholder="Postal Code" defaultValue="A1A 1A1"/>
      <Select
        value={country.code}
        onValueChange={(value) => {
          const next = uniqueCountries.find((c) => c.code === value)
          if (next) setCountry(next)
        }}
      >
        <SelectTrigger aria-label="Country" className="col-span-2">
          <SelectValue placeholder="Country"/>
        </SelectTrigger>
        <SelectContent>
          {uniqueCountries.map((country) => (
            <SelectItem key={country.code} value={country.code}>
              <span className="w-5"> {country.flag} </span>{" "}
              <span>{country.name}</span>
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}
